package service

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"sort"
	"sync"
	"sync/atomic"
	"time"
	"unicode/utf8"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/setting/ratio_setting"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

const (
	ModelPerformanceBucketSeconds = int64(300)
	perfFlushEvery                = 30 * time.Second
	perfMaxPending                = 8192
	perfMaxModels                 = 2000
)

type perfOutcome uint8

const (
	perfOtherError perfOutcome = iota
	perfCompleted
	perfLimited
	perfCanceled
	perfRejected
	perfNoRoute
	perfGatewayLimit
	perfUpstream429
	perfUpstream5xx
	perfStreamError
)

type perfObservation struct {
	outcome      perfOutcome
	durationMS   int64
	firstMS      int64
	hasFirst     bool
	tokens       int64
	generationMS int64
}

var perfFirstBounds = [...]int64{100, 250, 500, 1_000, 2_000, 3_000, 5_000, 8_000, 12_000, 20_000, 30_000, 60_000, 120_000}

func perfFirstCounts(c *model.ModelPerformanceCounts) [14]*int64 {
	return [14]*int64{&c.First100, &c.First250, &c.First500, &c.First1000, &c.First2000, &c.First3000,
		&c.First5000, &c.First8000, &c.First12000, &c.First20000, &c.First30000, &c.First60000, &c.First120000, &c.FirstOver}
}

func addPerfObservation(c *model.ModelPerformanceCounts, obs perfObservation) {
	c.Samples++
	switch obs.outcome {
	case perfCompleted:
		c.Completed++
	case perfLimited:
		c.Limited++
	case perfCanceled:
		c.Canceled++
	case perfRejected:
		c.Rejected++
	case perfNoRoute:
		c.NoRoute++
	case perfGatewayLimit:
		c.GatewayLimit++
	case perfUpstream429:
		c.Upstream429++
	case perfUpstream5xx:
		c.Upstream5xx++
	case perfStreamError:
		c.StreamError++
	default:
		c.OtherError++
	}
	if obs.outcome != perfCompleted && obs.outcome != perfLimited {
		return
	}
	if obs.durationMS >= 0 {
		c.DurationMS += obs.durationMS
		c.DurationN++
	}
	if obs.hasFirst && obs.firstMS >= 0 {
		bins := perfFirstCounts(c)
		i := sort.Search(len(perfFirstBounds), func(i int) bool { return obs.firstMS <= perfFirstBounds[i] })
		*bins[i]++
	}
	if obs.tokens > 0 && obs.tokens <= 16*1024*1024 && obs.generationMS > 0 {
		c.OutputTokens += obs.tokens
		c.GenerationMS += obs.generationMS
		c.GenerationN++
	}
}

type perfModel struct{ key, name string }

func validPerfModel(name string) bool {
	if name == "" || len(name) > 128 || !utf8.ValidString(name) {
		return false
	}
	for _, char := range name {
		if char <= 0x20 || char == 0x7f {
			return false
		}
	}
	return true
}

func perfModelID(name string) perfModel {
	sum := sha256.Sum256([]byte(name))
	return perfModel{key: hex.EncodeToString(sum[:]), name: name}
}

type perfKey struct {
	ts      int64
	model   string
	channel int
}

type perfPending struct {
	row   model.ModelPerformanceBucket
	saved int64
}

type performanceCollector struct {
	db           *gorm.DB
	process      string
	catalog      atomic.Pointer[map[string]perfModel]
	dropped      atomic.Int64
	writeDelayed atomic.Bool
	mu           sync.Mutex
	latest       int64
	pending      map[perfKey]*perfPending
}

var activePerformance atomic.Pointer[performanceCollector]

func newPerformanceCollector(db *gorm.DB) *performanceCollector {
	return &performanceCollector{db: db, process: uuid.NewString(), pending: make(map[perfKey]*perfPending)}
}

func (c *performanceCollector) refreshCatalog(ctx context.Context) error {
	names, err := model.ModelPerformanceCatalog(ctx, c.db, perfMaxModels)
	if err != nil {
		return err
	}
	catalog := make(map[string]perfModel, len(names))
	for _, name := range names {
		if validPerfModel(name) {
			catalog[name] = perfModelID(name)
		}
	}
	c.catalog.Store(&catalog)
	return nil
}

func (c *performanceCollector) lookup(name string) perfModel {
	catalog := c.catalog.Load()
	if catalog == nil || !validPerfModel(name) {
		return perfModel{}
	}
	if match, ok := (*catalog)[name]; ok {
		return match
	}
	return (*catalog)[ratio_setting.FormatMatchingModelName(name)]
}

func (c *performanceCollector) record(id perfModel, channel int, obs perfObservation, now time.Time) {
	if id.key == "" || channel < 0 {
		return
	}
	ts := now.Unix()
	c.mu.Lock()
	defer c.mu.Unlock()
	// A late observation or a backwards system-clock adjustment must not
	// recreate a retired snapshot and overwrite a larger persisted count.
	c.latest = max(c.latest, ts-ts%ModelPerformanceBucketSeconds)
	key := perfKey{ts: c.latest, model: id.key, channel: channel}
	bucket := c.pending[key]
	if bucket == nil {
		if len(c.pending) >= perfMaxPending {
			c.dropped.Add(1)
			return
		}
		bucket = &perfPending{row: model.ModelPerformanceBucket{
			ProcessID: c.process, BucketStart: key.ts, ModelKey: id.key, ModelName: id.name, ChannelID: channel,
		}}
		c.pending[key] = bucket
	}
	bucket.row.LastSeen = max(ts, bucket.row.LastSeen)
	addPerfObservation(&bucket.row.ModelPerformanceCounts, obs)
}

// The single worker owns all flushes. No database or Redis I/O happens on a
// relay request; the lock only protects counters and a bounded snapshot copy.
func (c *performanceCollector) flush(ctx context.Context, now time.Time) error {
	c.mu.Lock()
	ts := now.Unix()
	c.latest = max(c.latest, ts-ts%ModelPerformanceBucketSeconds)
	rows := make([]model.ModelPerformanceBucket, 0, len(c.pending))
	for key, b := range c.pending {
		if b.saved < b.row.Samples {
			rows = append(rows, b.row)
		} else if key.ts < c.latest {
			delete(c.pending, key)
		}
	}
	c.mu.Unlock()
	if err := model.StoreModelPerformance(ctx, c.db, rows); err != nil {
		return err
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	for _, row := range rows {
		key := perfKey{ts: row.BucketStart, model: row.ModelKey, channel: row.ChannelID}
		b := c.pending[key]
		b.saved = row.Samples
		if key.ts < c.latest && b.row.Samples == b.saved {
			delete(c.pending, key)
		}
	}
	return nil
}

// StartModelPerformance keeps optional telemetry startup and migration out of
// server readiness. Workers sharing a DB aggregate without double-adding.
func StartModelPerformance() func() {
	if !common.GetEnvOrDefaultBool("MODEL_PERFORMANCE_ENABLED", true) || model.DB == nil {
		return func() {}
	}
	c := newPerformanceCollector(model.DB)
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	go func() {
		defer close(done)
		if common.IsMasterNode {
			migration, stop := context.WithTimeout(ctx, 8*time.Second)
			err := c.db.WithContext(migration).AutoMigrate(&model.ModelPerformanceBucket{})
			stop()
			if err != nil {
				common.SysError("model performance disabled: could not initialize metric storage")
				return
			}
		}
		catalog, stop := context.WithTimeout(ctx, 3*time.Second)
		if err := c.refreshCatalog(catalog); err != nil {
			common.SysError("model performance initial catalog unavailable")
		}
		stop()
		activePerformance.Store(c)
		defer activePerformance.CompareAndSwap(c, nil)
		flush := time.NewTicker(perfFlushEvery)
		defer flush.Stop()
		refresh := time.NewTicker(2 * time.Minute)
		defer refresh.Stop()
		prune := time.NewTicker(15 * time.Minute)
		defer prune.Stop()
		for {
			select {
			case <-ctx.Done():
				stopCtx, stop := context.WithTimeout(context.Background(), 3*time.Second)
				if err := c.flush(stopCtx, time.Now()); err != nil {
					common.SysError("model performance shutdown snapshot not saved")
				}
				stop()
				return
			case now := <-flush.C:
				flushCtx, stop := context.WithTimeout(ctx, 5*time.Second)
				err := c.flush(flushCtx, now)
				c.writeDelayed.Store(err != nil)
				if err != nil {
					common.SysError("model performance snapshot not saved; will retry")
				}
				stop()
			case <-refresh.C:
				refreshCtx, stop := context.WithTimeout(ctx, 3*time.Second)
				if err := c.refreshCatalog(refreshCtx); err != nil {
					common.SysError("model performance catalog refresh failed")
				}
				stop()
			case now := <-prune.C:
				if common.IsMasterNode {
					pruneCtx, stop := context.WithTimeout(ctx, 4*time.Second)
					if err := model.PruneModelPerformance(pruneCtx, c.db, now.Add(-7*24*time.Hour).Unix()); err != nil {
						common.SysError("model performance retention pass failed")
					}
					stop()
				}
			}
		}
	}()
	return func() {
		cancel()
		select {
		case <-done:
		case <-time.After(4 * time.Second):
		}
	}
}
