package model

import (
	"context"
	"slices"
	"strings"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// ModelPerformanceBucket is an absolute snapshot for one process, five-minute
// bucket, model and metric scope. A channel of zero denotes final requests,
// not the sum of attempts. Retrying a flush overwrites, never adds twice.
type ModelPerformanceBucket struct {
	Id                     int64  `gorm:"primaryKey"`
	ProcessID              string `gorm:"size:36;not null;uniqueIndex:idx_modelperf_snapshot,priority:1"`
	BucketStart            int64  `gorm:"not null;uniqueIndex:idx_modelperf_snapshot,priority:2;index:idx_modelperf_time;index:idx_modelperf_detail,priority:2"`
	ModelKey               string `gorm:"size:64;not null;uniqueIndex:idx_modelperf_snapshot,priority:3;index:idx_modelperf_detail,priority:1"`
	ModelName              string `gorm:"size:128;not null"`
	ChannelID              int    `gorm:"not null;uniqueIndex:idx_modelperf_snapshot,priority:4"`
	LastSeen               int64  `gorm:"not null"`
	ModelPerformanceCounts `gorm:"embedded"`
}

type ModelPerformanceCounts struct {
	Samples      int64
	Completed    int64
	Limited      int64
	Canceled     int64
	Rejected     int64
	NoRoute      int64
	GatewayLimit int64
	Upstream429  int64 `gorm:"column:upstream_429"`
	Upstream5xx  int64 `gorm:"column:upstream_5xx"`
	StreamError  int64
	OtherError   int64
	DurationMS   int64 `gorm:"column:duration_ms"`
	DurationN    int64
	OutputTokens int64
	GenerationMS int64 `gorm:"column:generation_ms"`
	GenerationN  int64
	First100     int64 `gorm:"column:first_100"`
	First250     int64 `gorm:"column:first_250"`
	First500     int64 `gorm:"column:first_500"`
	First1000    int64 `gorm:"column:first_1000"`
	First2000    int64 `gorm:"column:first_2000"`
	First3000    int64 `gorm:"column:first_3000"`
	First5000    int64 `gorm:"column:first_5000"`
	First8000    int64 `gorm:"column:first_8000"`
	First12000   int64 `gorm:"column:first_12000"`
	First20000   int64 `gorm:"column:first_20000"`
	First30000   int64 `gorm:"column:first_30000"`
	First60000   int64 `gorm:"column:first_60000"`
	First120000  int64 `gorm:"column:first_120000"`
	FirstOver    int64 `gorm:"column:first_over"`
}

// Each counter is additive across independent processes and time buckets.
var modelPerformanceCounters = []string{
	"samples", "completed", "limited", "canceled", "rejected", "no_route",
	"gateway_limit", "upstream_429", "upstream_5xx", "stream_error", "other_error",
	"duration_ms", "duration_n", "output_tokens", "generation_ms", "generation_n",
	"first_100", "first_250", "first_500", "first_1000", "first_2000", "first_3000",
	"first_5000", "first_8000", "first_12000", "first_20000", "first_30000", "first_60000",
	"first_120000", "first_over",
}

func StoreModelPerformance(ctx context.Context, db *gorm.DB, rows []ModelPerformanceBucket) error {
	if len(rows) == 0 {
		return nil
	}
	columns := append([]string{"last_seen"}, modelPerformanceCounters...)
	return db.WithContext(ctx).Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "process_id"}, {Name: "bucket_start"}, {Name: "model_key"}, {Name: "channel_id"}},
		DoUpdates: clause.AssignmentColumns(columns),
	}).CreateInBatches(&rows, 20).Error
}

type ModelPerformanceRollup struct {
	ModelKey               string
	ModelName              string
	ChannelID              int
	BucketStart            int64
	LastSeen               int64
	ModelPerformanceCounts `gorm:"embedded"`
}

var modelPerformanceSums = func() string {
	selects := []string{"MAX(last_seen) AS last_seen"}
	for _, name := range modelPerformanceCounters {
		selects = append(selects, "SUM("+name+") AS "+name)
	}
	return strings.Join(selects, ", ")
}()

func ReadModelPerformance(ctx context.Context, db *gorm.DB, from, to int64, key string, details bool) ([]ModelPerformanceRollup, error) {
	q := db.WithContext(ctx).Model(&ModelPerformanceBucket{}).
		Where("bucket_start >= ? AND bucket_start <= ?", from, to)
	if key != "" {
		q = q.Where("model_key = ?", key)
	}
	if details {
		q = q.Where("channel_id > 0").Select("channel_id, " + modelPerformanceSums).Group("channel_id")
	} else {
		q = q.Where("channel_id = 0").Select("model_key, model_name, " + modelPerformanceSums).Group("model_key, model_name")
	}
	var rows []ModelPerformanceRollup
	err := q.Scan(&rows).Error
	return rows, err
}

func ReadModelPerformanceSeries(ctx context.Context, db *gorm.DB, from, to int64, key string) ([]ModelPerformanceRollup, error) {
	var rows []ModelPerformanceRollup
	err := db.WithContext(ctx).Model(&ModelPerformanceBucket{}).
		Where("bucket_start >= ? AND bucket_start <= ? AND model_key = ? AND channel_id = 0", from, to, key).
		Select("bucket_start, " + modelPerformanceSums).Group("bucket_start").Order("bucket_start").Scan(&rows).Error
	return rows, err
}

func ModelPerformanceCatalog(ctx context.Context, db *gorm.DB, limit int) ([]string, error) {
	var names []string
	err := db.WithContext(ctx).Model(&Ability{}).Where("enabled = ?", true).Distinct("model").Order("model").Limit(limit).Pluck("model", &names).Error
	return names, err
}

func ModelPerformanceChannelNames(ctx context.Context, db *gorm.DB, ids []int) (map[int]string, error) {
	names := make(map[int]string, len(ids))
	for chunk := range slices.Chunk(ids, 250) {
		var rows []struct {
			Id   int
			Name string
		}
		if err := db.WithContext(ctx).Model(&Channel{}).Select("id", "name").Where("id IN ?", chunk).Scan(&rows).Error; err != nil {
			return nil, err
		}
		for _, r := range rows {
			names[r.Id] = r.Name
		}
	}
	return names, nil
}

// Prune only the dedicated metric table in small chunks; never touch request
// logs or billing data. The ID list also works on engines without DELETE LIMIT.
func PruneModelPerformance(ctx context.Context, db *gorm.DB, before int64) error {
	for range 4 {
		var ids []int64
		if err := db.WithContext(ctx).Model(&ModelPerformanceBucket{}).Where("bucket_start < ?", before).
			Order("bucket_start, id").Limit(500).Pluck("id", &ids).Error; err != nil {
			return err
		}
		if len(ids) == 0 {
			return nil
		}
		if err := db.WithContext(ctx).Where("id IN ? AND bucket_start < ?", ids, before).Delete(&ModelPerformanceBucket{}).Error; err != nil {
			return err
		}
		if len(ids) < 500 {
			return nil
		}
	}
	return nil
}
