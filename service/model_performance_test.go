package service

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/QuantumNous/new-api/model"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	"github.com/QuantumNous/new-api/types"
	"github.com/gin-gonic/gin"
	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func perfTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(filepath.Join(t.TempDir(), "metrics.db")), &gorm.Config{})
	require.NoError(t, err)
	sql, err := db.DB()
	require.NoError(t, err)
	sql.SetMaxOpenConns(1)
	t.Cleanup(func() { _ = sql.Close() })
	require.NoError(t, db.AutoMigrate(&model.ModelPerformanceBucket{}, &model.Ability{}, &model.Channel{}))
	return db
}

func perfFixture(t *testing.T, db *gorm.DB) *performanceCollector {
	t.Helper()
	for _, name := range []string{"fixture-model", "idle-model"} {
		require.NoError(t, db.Create(&model.Ability{Group: "default", Model: name, ChannelId: 20, Enabled: true}).Error)
	}
	c := newPerformanceCollector(db)
	require.NoError(t, c.refreshCatalog(context.Background()))
	require.True(t, activePerformance.CompareAndSwap(nil, c))
	t.Cleanup(func() { activePerformance.CompareAndSwap(c, nil) })
	return c
}

func perfTestContext() *gin.Context {
	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	c.Request = httptest.NewRequest(http.MethodPost, "/v1/chat/completions", nil)
	return c
}

func upstreamPerformanceError(status int) *types.NewAPIError {
	return types.WithOpenAIError(types.OpenAIError{Message: "fixture", Code: "fixture"}, status, types.ErrOptionWithUpstreamResponse())
}

func TestModelPerformanceSeparatesRetryAttemptsFromFinalRequests(t *testing.T) {
	db := perfTestDB(t)
	col := perfFixture(t, db)
	ctx := context.Background()
	c := perfTestContext()
	trace := BeginModelPerformance(c)
	require.NotNil(t, trace)
	SetModelPerformanceModel(c, "fixture-model")
	since := time.Now()
	trace.ObserveAttempt(c, nil, 20, upstreamPerformanceError(http.StatusTooManyRequests), since, since)
	trace.ObserveAttempt(c, &relaycommon.RelayInfo{}, 21, nil, since, since)
	trace.SetRelayFinal(c, &relaycommon.RelayInfo{}, nil)
	trace.Finish(c)

	c = perfTestContext()
	noRoute := BeginModelPerformance(c)
	SetModelPerformanceModel(c, "fixture-model")
	MarkModelPerformanceNoRoute(c)
	c.Status(http.StatusServiceUnavailable)
	noRoute.Finish(c)

	c = perfTestContext()
	unknown := BeginModelPerformance(c)
	SetModelPerformanceModel(c, "unknown-body-value")
	c.Status(http.StatusServiceUnavailable)
	unknown.Finish(c)

	now := time.Now()
	require.NoError(t, col.flush(ctx, now))
	require.NoError(t, col.flush(ctx, now))
	got, err := ReadModelPerformanceOverview(ctx, 60, "fixture-model", now)
	require.NoError(t, err)
	require.True(t, got.Ready)
	require.Len(t, got.Models, 1)
	require.EqualValues(t, 2, got.Models[0].Samples)
	require.EqualValues(t, 1, got.Models[0].Completed)
	require.EqualValues(t, 1, got.Models[0].NoRoute)
	require.Zero(t, got.Models[0].Upstream429)
	require.InDelta(t, 0.5, *got.Models[0].CompletionRate, 0.001)
	require.Len(t, got.Channels, 2)
	require.Equal(t, 20, got.Channels[0].ChannelID)
	require.EqualValues(t, 1, got.Channels[0].Upstream429)
	require.EqualValues(t, 1, got.Channels[1].Completed)
	require.NotEmpty(t, got.Series)
	all, err := ReadModelPerformanceOverview(ctx, 60, "", now)
	require.NoError(t, err)
	require.Len(t, all.Models, 2)
	require.Equal(t, "idle-model", all.Models[1].Model)
	require.Nil(t, all.Models[1].CompletionRate)
	require.Nil(t, all.Models[1].GenerationTPS)
	require.EqualValues(t, 0, all.Models[1].Samples)
}

func TestModelPerformanceSnapshotIsIdempotentAndKeepsOtherProcesses(t *testing.T) {
	db := perfTestDB(t)
	a, b := newPerformanceCollector(db), newPerformanceCollector(db)
	now := time.Now()
	id := perfModelID("fixture-model")
	observation := perfObservation{outcome: perfCompleted, durationMS: 2_000, hasFirst: true, firstMS: 350, tokens: 80, generationMS: 800}
	var wg sync.WaitGroup
	for range 8 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for range 25 {
				a.record(id, 0, observation, now)
			}
		}()
	}
	wg.Wait()
	b.record(id, 0, observation, now)
	ctx := context.Background()
	require.NoError(t, a.flush(ctx, now))
	require.NoError(t, b.flush(ctx, now))
	a.record(id, 0, observation, now)
	require.NoError(t, a.flush(ctx, now))
	a.mu.Lock()
	for _, p := range a.pending {
		p.saved = 0 // a flush committed but its local acknowledgement was lost
	}
	a.mu.Unlock()
	require.NoError(t, a.flush(ctx, now))
	rows, err := model.ReadModelPerformance(ctx, db, now.Add(-5*time.Minute).Unix(), now.Unix(), id.key, false)
	require.NoError(t, err)
	require.Len(t, rows, 1)
	row := modelPerformanceRow(rows[0])
	require.EqualValues(t, 202, row.Samples)
	require.EqualValues(t, 202, row.FirstSamples)
	require.EqualValues(t, 500, *row.FirstP95MS)
	require.EqualValues(t, 100, *row.GenerationTPS)
	require.EqualValues(t, 2_000, *row.MeanDurationMS)
}

func TestModelPerformanceClassifiesStreamTermination(t *testing.T) {
	done := relaycommon.NewStreamStatus()
	done.SetEndReason(relaycommon.StreamEndReasonDone, nil)
	eof := relaycommon.NewStreamStatus()
	eof.SetEndReason(relaycommon.StreamEndReasonEOF, nil)
	gone := relaycommon.NewStreamStatus()
	gone.SetEndReason(relaycommon.StreamEndReasonClientGone, nil)
	normal := &relaycommon.RelayInfo{IsStream: true, StreamStatus: done}
	assert := func(expected perfOutcome, info *relaycommon.RelayInfo, err *types.NewAPIError) {
		require.Equal(t, expected, classifyPerformance(info, err, false))
	}
	assert(perfCompleted, normal, nil)
	assert(perfStreamError, &relaycommon.RelayInfo{IsStream: true, StreamStatus: eof}, nil)
	assert(perfStreamError, &relaycommon.RelayInfo{IsStream: true}, nil)
	assert(perfCanceled, &relaycommon.RelayInfo{IsStream: true, StreamStatus: gone}, nil)
	assert(perfLimited, &relaycommon.RelayInfo{IsStream: true, StreamStatus: done,
		ResponsesOutcome: &relaycommon.ResponsesOutcome{State: "incomplete", Reason: "max_output_tokens"}}, nil)
	assert(perfRejected, nil, types.NewErrorWithStatusCode(errors.New("fixture"), types.ErrorCodeBadRequestBody, 400))
	assert(perfUpstream429, nil, upstreamPerformanceError(429))
	assert(perfUpstream5xx, nil, upstreamPerformanceError(503))
	assert(perfOtherError, nil, upstreamPerformanceError(401))
}

func TestModelPerformanceP95AndEmptyDenominators(t *testing.T) {
	row := modelPerformanceRow(model.ModelPerformanceRollup{ModelPerformanceCounts: model.ModelPerformanceCounts{
		Samples: 22, Canceled: 1, Rejected: 1, Completed: 19, Limited: 1,
		First100: 18, First250: 1, FirstOver: 1,
	}})
	require.EqualValues(t, 20, row.Eligible)
	require.EqualValues(t, 250, *row.FirstP95MS)
	require.False(t, row.FirstP95Over)
	require.InDelta(t, 0.95, *row.CompletionRate, .001)
	require.Nil(t, row.GenerationTPS)
	slow := modelPerformanceRow(model.ModelPerformanceRollup{ModelPerformanceCounts: model.ModelPerformanceCounts{Samples: 1, FirstOver: 1}})
	require.True(t, slow.FirstP95Over)
	require.EqualValues(t, 120000, *slow.FirstP95MS)
}
