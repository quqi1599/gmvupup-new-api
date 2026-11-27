package service

import (
	"context"
	"errors"
	"sort"
	"time"

	"github.com/QuantumNous/new-api/model"
)

var ErrInvalidModelPerformanceModel = errors.New("invalid model performance filter")

type ModelPerformanceRow struct {
	Model          string   `json:"model,omitempty"`
	ChannelID      int      `json:"channel_id,omitempty"`
	ChannelName    string   `json:"channel_name,omitempty"`
	BucketStart    int64    `json:"bucket_start,omitempty"`
	LastSeen       int64    `json:"last_seen"`
	Samples        int64    `json:"samples"`
	Completed      int64    `json:"completed"`
	Limited        int64    `json:"limited"`
	Canceled       int64    `json:"canceled"`
	Rejected       int64    `json:"rejected"`
	NoRoute        int64    `json:"no_route"`
	GatewayLimit   int64    `json:"gateway_limit"`
	Upstream429    int64    `json:"upstream_429"`
	Upstream5xx    int64    `json:"upstream_5xx"`
	StreamError    int64    `json:"stream_error"`
	OtherError     int64    `json:"other_error"`
	Eligible       int64    `json:"eligible"`
	CompletionRate *float64 `json:"completion_rate"`
	MeanDurationMS *float64 `json:"mean_duration_ms"`
	FirstP95MS     *int64   `json:"first_p95_ms"`
	FirstP95Over   bool     `json:"first_p95_over"`
	FirstSamples   int64    `json:"first_samples"`
	GenerationTPS  *float64 `json:"generation_tps"`
	SpeedSamples   int64    `json:"speed_samples"`
}

type ModelPerformanceOverview struct {
	Ready         bool                  `json:"ready"`
	WriteDelayed  bool                  `json:"write_delayed"`
	Dropped       int64                 `json:"dropped_on_this_node"`
	From          int64                 `json:"from"`
	To            int64                 `json:"to"`
	BucketSeconds int64                 `json:"bucket_seconds"`
	Models        []ModelPerformanceRow `json:"models"`
	Channels      []ModelPerformanceRow `json:"channels"`
	Series        []ModelPerformanceRow `json:"series"`
}

func modelPerformanceRow(r model.ModelPerformanceRollup) ModelPerformanceRow {
	c := r.ModelPerformanceCounts
	row := ModelPerformanceRow{
		Model: r.ModelName, ChannelID: r.ChannelID, BucketStart: r.BucketStart, LastSeen: r.LastSeen,
		Samples: c.Samples, Completed: c.Completed, Limited: c.Limited, Canceled: c.Canceled,
		Rejected: c.Rejected, NoRoute: c.NoRoute, GatewayLimit: c.GatewayLimit,
		Upstream429: c.Upstream429, Upstream5xx: c.Upstream5xx, StreamError: c.StreamError, OtherError: c.OtherError,
		Eligible: c.Samples - c.Canceled - c.Rejected, SpeedSamples: c.GenerationN,
	}
	if row.Eligible > 0 {
		rate := float64(c.Completed) / float64(row.Eligible)
		row.CompletionRate = &rate
	}
	if c.DurationN > 0 {
		avg := float64(c.DurationMS) / float64(c.DurationN)
		row.MeanDurationMS = &avg
	}
	if c.GenerationN > 0 && c.GenerationMS > 0 {
		tps := 1000 * float64(c.OutputTokens) / float64(c.GenerationMS)
		row.GenerationTPS = &tps
	}
	bins := perfFirstCounts(&c)
	for _, n := range bins {
		row.FirstSamples += *n
	}
	if row.FirstSamples > 0 {
		rank := row.FirstSamples - row.FirstSamples/20 // ceil(0.95 * N)
		var cumulative int64
		for i, n := range bins {
			cumulative += *n
			if cumulative >= rank {
				bound := perfFirstBounds[min(i, len(perfFirstBounds)-1)]
				row.FirstP95MS = &bound
				row.FirstP95Over = i == len(perfFirstBounds)
				break
			}
		}
	}
	return row
}

func ReadModelPerformanceOverview(ctx context.Context, minutes int, selected string, now time.Time) (ModelPerformanceOverview, error) {
	if minutes != 15 && minutes != 60 && minutes != 1440 || selected != "" && !validPerfModel(selected) {
		return ModelPerformanceOverview{}, ErrInvalidModelPerformanceModel
	}
	end := now.Unix()
	start := end - end%ModelPerformanceBucketSeconds - int64(minutes*60) + ModelPerformanceBucketSeconds
	data := ModelPerformanceOverview{
		From: start, To: end, BucketSeconds: ModelPerformanceBucketSeconds,
		Models: []ModelPerformanceRow{}, Channels: []ModelPerformanceRow{}, Series: []ModelPerformanceRow{},
	}
	col := activePerformance.Load()
	if col == nil {
		return data, nil
	}
	catalog := col.catalog.Load()
	if catalog == nil {
		return data, nil
	}
	data.Ready, data.Dropped, data.WriteDelayed = true, col.dropped.Load(), col.writeDelayed.Load()
	var key string
	if selected != "" {
		key = perfModelID(selected).key
	}
	rows, err := model.ReadModelPerformance(ctx, col.db, start, end, key, false)
	if err != nil {
		return data, err
	}
	seen := make(map[string]bool, len(rows))
	for _, r := range rows {
		seen[r.ModelKey] = true
		data.Models = append(data.Models, modelPerformanceRow(r))
	}
	for _, id := range *catalog {
		if !seen[id.key] && (key == "" || id.key == key) {
			data.Models = append(data.Models, ModelPerformanceRow{Model: id.name})
		}
	}
	sort.Slice(data.Models, func(i, j int) bool {
		if data.Models[i].Samples != data.Models[j].Samples {
			return data.Models[i].Samples > data.Models[j].Samples
		}
		return data.Models[i].Model < data.Models[j].Model
	})
	if selected == "" || len(data.Models) == 0 || data.Models[0].Samples == 0 {
		return data, nil
	}
	attempts, err := model.ReadModelPerformance(ctx, col.db, start, end, key, true)
	if err != nil {
		return data, err
	}
	ids := make([]int, 0, len(attempts))
	for _, channel := range attempts {
		ids = append(ids, channel.ChannelID)
	}
	names, err := model.ModelPerformanceChannelNames(ctx, col.db, ids)
	if err != nil {
		return data, err
	}
	for _, r := range attempts {
		row := modelPerformanceRow(r)
		row.ChannelName = names[row.ChannelID]
		data.Channels = append(data.Channels, row)
	}
	sort.Slice(data.Channels, func(i, j int) bool {
		if data.Channels[i].Samples != data.Channels[j].Samples {
			return data.Channels[i].Samples > data.Channels[j].Samples
		}
		return data.Channels[i].ChannelID < data.Channels[j].ChannelID
	})
	series, err := model.ReadModelPerformanceSeries(ctx, col.db, start, end, key)
	if err != nil {
		return data, err
	}
	for _, r := range series {
		data.Series = append(data.Series, modelPerformanceRow(r))
	}
	return data, nil
}
