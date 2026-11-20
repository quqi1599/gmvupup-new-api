package controller

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/QuantumNous/new-api/relay/helper"
	"github.com/QuantumNous/new-api/types"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func TestWriteRelayErrorSuppressesJSONAfterSSEOutput(t *testing.T) {
	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	c.Request = httptest.NewRequest(http.MethodPost, "/v1/chat/completions", nil)
	require.NoError(t, helper.StringData(c, `{"ok":true}`))
	before := recorder.Body.String()

	relayErr := types.NewOpenAIError(errors.New("upstream stream failed"), types.ErrorCodeBadResponse, http.StatusBadGateway)
	writeRelayError(c, nil, types.RelayFormatOpenAI, relayErr)

	require.Equal(t, before, recorder.Body.String())
	require.NotContains(t, recorder.Body.String(), `"error"`)
}

func TestWriteRelayErrorRestoresJSONHeadersBeforeFirstOutput(t *testing.T) {
	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	c.Request = httptest.NewRequest(http.MethodPost, "/v1/chat/completions", nil)
	helper.SetEventStreamHeaders(c)

	relayErr := types.NewOpenAIError(errors.New("upstream stream failed"), types.ErrorCodeBadResponse, http.StatusBadGateway)
	writeRelayError(c, nil, types.RelayFormatOpenAI, relayErr)

	require.Equal(t, http.StatusBadGateway, recorder.Code)
	require.Equal(t, "application/json; charset=utf-8", recorder.Header().Get("Content-Type"))
	require.NotContains(t, recorder.Header().Get("Content-Type"), "text/event-stream")
	require.Contains(t, recorder.Body.String(), `"error"`)
}

func TestWriteRelayErrorAddsRetryAfterForAuthUnavailable(t *testing.T) {
	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	c.Request = httptest.NewRequest(http.MethodPost, "/v1/responses", nil)
	relayErr := types.WithOpenAIError(
		types.OpenAIError{
			Message: "requested route is temporarily unavailable",
			Type:    "upstream_error",
			Code:    string(types.ErrorCodeAuthUnavailable),
		},
		http.StatusServiceUnavailable,
		types.ErrOptionWithUpstreamResponse(),
	)

	writeRelayError(c, nil, types.RelayFormatOpenAI, relayErr)

	require.Equal(t, authUnavailableRetryAfterSeconds, recorder.Header().Get("Retry-After"))
	require.Equal(t, http.StatusServiceUnavailable, recorder.Code)
}

func TestWriteRelayErrorDoesNotAddRetryAfterForOtherErrors(t *testing.T) {
	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	c.Request = httptest.NewRequest(http.MethodPost, "/v1/responses", nil)
	relayErr := types.NewOpenAIError(errors.New("upstream failed"), types.ErrorCodeBadResponse, http.StatusBadGateway)

	writeRelayError(c, nil, types.RelayFormatOpenAI, relayErr)

	require.Empty(t, recorder.Header().Get("Retry-After"))
}

func TestWriteRelayErrorPreservesRetryAfterForCompactionRouteUnavailable(t *testing.T) {
	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	c.Request = httptest.NewRequest(http.MethodPost, "/v1/responses", nil)
	relayErr := types.WithOpenAIError(
		types.OpenAIError{
			Message: "all compatible remote-compaction routes are temporarily unavailable",
			Type:    "server_error",
			Code:    string(types.ErrorCodeCompactionRouteUnavailable),
		},
		http.StatusServiceUnavailable,
		types.ErrOptionWithUpstreamResponse(),
	)
	relayErr.SetRetryAfter("17")

	writeRelayError(c, nil, types.RelayFormatOpenAI, relayErr)

	require.Equal(t, "17", recorder.Header().Get("Retry-After"))
	require.Equal(t, http.StatusServiceUnavailable, recorder.Code)
	require.Contains(t, recorder.Body.String(), string(types.ErrorCodeCompactionRouteUnavailable))
}

func TestWriteChatCooldownWithUpstreamWait(t *testing.T) {
	const date = "Sun, 06 Nov 1994 08:49:37 GMT"
	for _, status := range []int{http.StatusTooManyRequests, http.StatusServiceUnavailable} {
		for _, tc := range []struct{ value, expected, hint string }{
			{"17", "17", "retry after 17 seconds"}, {date, date, "retry after " + date},
			{"", "", ""}, {"-1", "", ""}, {"30s", "", ""}, {"1.5", "", ""}, {"3\r\nX:4", "", ""},
		} {
			recorder := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(recorder)
			c.Request = httptest.NewRequest(http.MethodPost, "/v1/chat/completions", nil)
			err := parseCPAContractError(t, status, `{"error":{"message":"eligible keys are waiting for recovery","type":"server_error","code":"model_cooldown"}}`)
			err.SetRetryAfter(tc.value)
			writeRelayError(c, nil, types.RelayFormatOpenAI, err)
			require.Equal(t, status, recorder.Code)
			require.Equal(t, tc.expected, recorder.Header().Get("Retry-After"))
			require.Contains(t, recorder.Body.String(), "model_cooldown")
			if tc.hint != "" {
				require.Contains(t, recorder.Body.String(), tc.hint)
			} else {
				require.NotContains(t, recorder.Body.String(), "retry after")
			}
		}
	}
}
