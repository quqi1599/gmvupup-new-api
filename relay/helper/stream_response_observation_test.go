package helper

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func TestStreamResponseObservationIsBoundedAndDoesNotChangeReads(t *testing.T) {
	body := `{"error":{"message":"BODY_SECRET"}}` + strings.Repeat("PRIVATE", 10000)
	obs := newStreamResponseObservation(&http.Response{StatusCode: 200, ContentLength: -1,
		Header: http.Header{"Content-Type": {"application/problem+json; private=HEADER_SECRET"}}}, true, "true")
	reader := io.TeeReader(strings.NewReader(body), obs)
	actual, err := io.ReadAll(reader)
	require.NoError(t, err)
	require.Equal(t, body, string(actual))
	require.Equal(t, responsePrefixLimit, obs.prefixLen)
	d := obs.snapshot()
	require.Equal(t, int64(len(body)), d.ReadBytes)
	require.Equal(t, "json_like", d.BodyPrefix)
	require.Equal(t, "application/json", d.ContentType)
	require.Equal(t, int64(-1), d.ContentLength)
	require.Zero(t, obs.prefixLen)
	require.Equal(t, [responsePrefixLimit]byte{}, obs.prefix)
	encoded, err := common.Marshal(d)
	require.NoError(t, err)
	for _, forbidden := range []string{"BODY_SECRET", "HEADER_SECRET", "PRIVATE", "message", "private="} {
		require.NotContains(t, string(encoded), forbidden)
	}
}

func TestStreamResponseObservationContentTypeAllowlist(t *testing.T) {
	for value, want := range map[string]string{
		"": "missing", "text/event-stream; boundary=SECRET": "text/event-stream",
		" TEXT/HTML ; charset=utf-8": "text/html", "application/json": "application/json",
		"text/plain": "text/plain", "application/x-SECRET": "other",
		strings.Repeat("SECRET", 100): "other",
	} {
		require.Equal(t, want, responseContentTypeClass(value))
	}
}

func TestRecordResponsesRequestStreamPreservesExplicitFalse(t *testing.T) {
	for _, tc := range []struct{ body, want string }{
		{`{}`, "omitted"}, {`{"stream":false}`, "false"}, {`{"stream":true}`, "true"},
		{`{"stream":null}`, "null"}, {`{"stream":"SECRET"}`, "non_boolean"},
		{`{"stream":0}`, "non_boolean"}, {`{"input":"SECRET","stream":{"SECRET":true}}`, "non_boolean"},
	} {
		c, _ := gin.CreateTestContext(httptest.NewRecorder())
		RecordResponsesRequestStream(c, []byte(tc.body))
		require.Equal(t, tc.want, common.GetContextKeyString(c, constant.ContextKeyUpstreamRequestStream))
	}
}
