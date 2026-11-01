package openai

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	"github.com/QuantumNous/new-api/relay/helper"
	"github.com/QuantumNous/new-api/service"
	"github.com/QuantumNous/new-api/types"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func TestResponsesPreEventFailureObservation(t *testing.T) {
	for _, tc := range []struct {
		name, contentType, body, prefix string
		bom                             bool
		data, comments, other           int64
		softErrors                      int
	}{
		{name: "empty", contentType: "text/event-stream", prefix: "empty"},
		{name: "json", contentType: "application/json; secret=HEADER_SECRET", body: `{"error":{"message":"BODY_SECRET"}}`, prefix: "json_like", other: 1},
		{name: "html", contentType: "text/html", body: "<!DOCTYPE html><html>BODY_SECRET</html>", prefix: "html_like", other: 1},
		{name: "comments", contentType: "text/event-stream", body: ": BODY_SECRET\n\n", prefix: "sse_like", comments: 1},
		{name: "BOM", contentType: "text/event-stream", body: "\xef\xbb\xbfdata: {\"type\":\"response.completed\"}\n\n", prefix: "sse_like", bom: true, other: 1},
		{name: "bad JSON frame", contentType: "text/event-stream", body: "data: BODY_SECRET\n\n", prefix: "sse_like", data: 1, softErrors: 1},
		{name: "missing type", contentType: "text/event-stream", body: "data: {\"message\":\"BODY_SECRET\"}\n\n", prefix: "sse_like", data: 1, softErrors: 1},
	} {
		t.Run(tc.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)
			c.Request = httptest.NewRequest(http.MethodPost, "/v1/responses", nil)
			helper.RecordResponsesRequestStream(c, []byte(`{"stream":true}`))
			info := &relaycommon.RelayInfo{IsStream: true, RelayFormat: types.RelayFormatOpenAIResponses,
				ChannelMeta: &relaycommon.ChannelMeta{UpstreamModelName: "deepseek-v4.1-flash"}}
			_, apiErr := OaiResponsesStreamHandler(c, info, &http.Response{StatusCode: 200,
				Header: http.Header{"Content-Type": {tc.contentType}}, ContentLength: int64(len(tc.body)),
				Body: io.NopCloser(strings.NewReader(tc.body))})
			require.NotNil(t, apiErr)
			require.Equal(t, 502, apiErr.StatusCode)
			require.Equal(t, types.ErrorCodeUpstreamStreamIncomplete, apiErr.GetErrorCode())
			require.True(t, types.IsSkipRetryError(apiErr), "diagnostics must not authorize another upstream call")
			require.Empty(t, w.Body.String())
			require.False(t, c.Writer.Written(), "do not manufacture an empty SSE success")
			d, ok := common.GetContextKeyType[*types.UpstreamResponseDiagnostic](c, constant.ContextKeyUpstreamResponseDiagnostic)
			require.True(t, ok)
			require.NotNil(t, d)
			require.Equal(t, int64(len(tc.body)), d.ReadBytes)
			require.Equal(t, int64(len(tc.body)), d.ContentLength)
			require.Equal(t, 200, d.StatusCode)
			require.True(t, d.StreamExpected)
			require.Equal(t, "true", d.RequestStream)
			require.Equal(t, tc.prefix, d.BodyPrefix)
			require.Equal(t, tc.bom, d.UTF8BOM)
			require.Equal(t, tc.data, d.DataLines)
			require.Equal(t, tc.comments, d.CommentLines)
			require.Equal(t, tc.other, d.OtherLines)
			require.Equal(t, "eof", d.EndReason)
			require.Equal(t, tc.softErrors, d.SoftErrors)
			admin := map[string]interface{}{}
			service.AppendRelayObservabilityAdminInfo(c, nil, admin)
			require.Equal(t, *d, admin["upstream_response"])
			serialized, err := common.Marshal(admin)
			require.NoError(t, err)
			require.NotContains(t, string(serialized), "BODY_SECRET")
			require.NotContains(t, string(serialized), "HEADER_SECRET")
		})
	}
}

func TestResponsesSuccessClearsPreviousObservation(t *testing.T) {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodPost, "/v1/responses", nil)
	common.SetContextKey(c, constant.ContextKeyUpstreamResponseDiagnostic, &types.UpstreamResponseDiagnostic{BodyPrefix: "stale"})
	info := &relaycommon.RelayInfo{IsStream: true, RelayFormat: types.RelayFormatOpenAIResponses,
		ChannelMeta: &relaycommon.ChannelMeta{UpstreamModelName: "deepseek-v4.1-flash"}}
	body := "data: {\"type\":\"response.completed\",\"response\":{\"id\":\"resp_test\",\"status\":\"completed\",\"output\":[],\"usage\":{\"input_tokens\":1,\"output_tokens\":1}}}\n\n"
	usage, apiErr := OaiResponsesStreamHandler(c, info, &http.Response{StatusCode: 200,
		Header: http.Header{"Content-Type": {"text/event-stream"}}, Body: io.NopCloser(strings.NewReader(body))})
	require.Nil(t, apiErr)
	require.NotNil(t, usage)
	require.Equal(t, 2, usage.TotalTokens)
	require.Contains(t, w.Body.String(), "response.completed")
	admin := map[string]interface{}{}
	service.AppendRelayObservabilityAdminInfo(c, info, admin)
	require.NotContains(t, admin, "upstream_response", "successful rounds must not inherit failed-round metadata")
}
