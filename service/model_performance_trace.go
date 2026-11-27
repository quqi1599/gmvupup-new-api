package service

import (
	"context"
	"errors"
	"net/http"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	"github.com/QuantumNous/new-api/types"
	"github.com/gin-gonic/gin"
)

const perfTraceKey = "model_performance_trace"

// ModelPerformanceTrace holds only request-scoped, numeric telemetry. The
// final request and its individual channel attempts have different grains.
type ModelPerformanceTrace struct {
	collector *performanceCollector
	model     perfModel
	started   time.Time
	final     *perfObservation
	noRoute   bool
	attempted bool
}

func BeginModelPerformance(c *gin.Context) *ModelPerformanceTrace {
	col := activePerformance.Load()
	if col == nil {
		return nil
	}
	t := &ModelPerformanceTrace{collector: col, started: time.Now()}
	c.Set(perfTraceKey, t)
	return t
}

func GetModelPerformanceTrace(c *gin.Context) *ModelPerformanceTrace {
	value, _ := c.Get(perfTraceKey)
	t, _ := value.(*ModelPerformanceTrace)
	if t != nil && t.model.key != "" {
		return t
	}
	return nil
}

func SetModelPerformanceModel(c *gin.Context, name string) {
	if v, ok := c.Get(perfTraceKey); ok {
		t := v.(*ModelPerformanceTrace)
		t.model = t.collector.lookup(name)
	}
}

func MarkModelPerformanceNoRoute(c *gin.Context) {
	if t := GetModelPerformanceTrace(c); t != nil {
		t.noRoute = true
	}
}

func classifyPerformance(info *relaycommon.RelayInfo, apiErr *types.NewAPIError, disconnected bool) perfOutcome {
	if apiErr != nil && (errors.Is(apiErr, context.Canceled) || disconnected && !apiErr.HasUpstreamResponse()) {
		return perfCanceled
	}
	if apiErr != nil && apiErr.HasUpstreamResponse() {
		switch {
		case apiErr.StatusCode == http.StatusTooManyRequests:
			return perfUpstream429
		case apiErr.StatusCode >= 500 && apiErr.StatusCode <= 599 && (info == nil || !info.HasSendResponse()):
			return perfUpstream5xx
		case apiErr.StatusCode == http.StatusUnauthorized || apiErr.StatusCode == http.StatusForbidden:
			return perfOtherError // provider authentication is not a client rejection
		}
	}
	if info != nil {
		ss := info.StreamStatus
		if info.IsStream && ss != nil &&
			(ss.EndReason == relaycommon.StreamEndReasonClientGone || ss.EndReason == relaycommon.StreamEndReasonPingFail) {
			return perfCanceled
		}
		if outcome := info.GenerationOutcome; outcome != nil {
			switch outcome.State {
			case "cancelled":
				return perfCanceled
			case "failed":
				if info.IsStream {
					return perfStreamError
				}
				return perfOtherError
			}
		}
		if info.IsStream && info.PartialStreamError != nil {
			return perfStreamError
		}
		if info.IsStream && ss != nil && (ss.HasErrors() ||
			ss.EndReason != relaycommon.StreamEndReasonNone && !ss.IsNormalEnd()) {
			return perfStreamError
		}
	}
	if apiErr != nil {
		if apiErr.StatusCode == http.StatusTooManyRequests {
			return perfGatewayLimit
		}
		if apiErr.StatusCode >= 400 && apiErr.StatusCode < 500 {
			return perfRejected
		}
		return perfOtherError
	}
	if info != nil {
		if info.IsResponsesIncomplete() {
			switch info.ResponsesOutcome.Reason {
			case "max_output_tokens", "max_tokens", "length":
				return perfLimited
			case "content_filter", "safety", "content_policy_violation":
				return perfRejected
			default:
				if info.IsStream {
					return perfStreamError
				}
				return perfOtherError
			}
		}
		// A 200 SSE header is not completion. When the adapter did not
		// observe a terminal event, keep this out of the success numerator.
		if info.IsStream && (info.StreamStatus == nil || !info.StreamStatus.IsNormalEnd()) {
			return perfStreamError
		}
	}
	return perfCompleted
}

func performanceObservation(info *relaycommon.RelayInfo, apiErr *types.NewAPIError, disconnected bool, start, end time.Time) perfObservation {
	result := perfObservation{outcome: classifyPerformance(info, apiErr, disconnected), durationMS: end.Sub(start).Milliseconds()}
	if info == nil || !info.IsStream || !info.HasSendResponse() ||
		info.FirstResponseTime.Before(start) || info.FirstResponseTime.After(end) {
		return result
	}
	result.hasFirst = true
	result.firstMS = info.FirstResponseTime.Sub(start).Milliseconds()
	result.tokens = info.PerformanceCompletionTokens
	result.generationMS = end.Sub(info.FirstResponseTime).Milliseconds()
	return result
}

func performanceDisconnected(c *gin.Context) bool {
	return common.GetContextKeyString(c, constant.ContextKeyRelayCancelOrigin) == constant.RelayCancelOriginDownstreamDisconnected
}

func (t *ModelPerformanceTrace) ObserveAttempt(c *gin.Context, info *relaycommon.RelayInfo, channel int, apiErr *types.NewAPIError, start, end time.Time) {
	t.attempted = true
	t.collector.record(t.model, channel, performanceObservation(info, apiErr, performanceDisconnected(c), start, end), end)
}

func (t *ModelPerformanceTrace) SetRelayFinal(c *gin.Context, info *relaycommon.RelayInfo, apiErr *types.NewAPIError) {
	result := performanceObservation(info, apiErr, performanceDisconnected(c), t.started, time.Now())
	if !t.attempted && apiErr != nil && !apiErr.HasUpstreamResponse() &&
		apiErr.StatusCode == http.StatusServiceUnavailable && apiErr.GetErrorCode() == types.ErrorCodeGetChannelFailed {
		result.outcome = perfNoRoute
	}
	t.final = &result
}

func (t *ModelPerformanceTrace) Finish(c *gin.Context) {
	if t.model.key == "" {
		return
	}
	now := time.Now()
	result := perfObservation{outcome: perfOtherError, durationMS: now.Sub(t.started).Milliseconds()}
	if t.final != nil {
		result = *t.final
	} else {
		switch {
		case performanceDisconnected(c) || c.Writer.Status() == 499:
			result.outcome = perfCanceled
		case t.noRoute:
			result.outcome = perfNoRoute
		case c.Writer.Status() == http.StatusTooManyRequests:
			result.outcome = perfGatewayLimit
		case c.Writer.Status() >= 400 && c.Writer.Status() < 500:
			result.outcome = perfRejected
		}
	}
	t.collector.record(t.model, 0, result, now)
}
