package gemini

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/dto"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	"github.com/QuantumNous/new-api/types"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func geminiFixture(t *testing.T, format types.RelayFormat, stream bool) (*gin.Context, *httptest.ResponseRecorder, *relaycommon.RelayInfo) {
	t.Helper()
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("POST", "/v1/test", nil)
	c.Set(common.RequestIdKey, "fixture")
	info := &relaycommon.RelayInfo{IsStream: stream, RelayFormat: format, OriginModelName: "test", ChannelMeta: &relaycommon.ChannelMeta{UpstreamModelName: "gemini-4-argon"}}
	if format == types.RelayFormatClaude {
		info.ClaudeConvertInfo = &relaycommon.ClaudeConvertInfo{}
	}
	return c, w, info
}

func geminiSSE(events ...string) *http.Response {
	var b strings.Builder
	for _, e := range events {
		b.WriteString("data: " + e + "\n\n")
	}
	return &http.Response{StatusCode: 200, Header: http.Header{"Content-Type": {"text/event-stream"}}, Body: io.NopCloser(strings.NewReader(b.String()))}
}

func TestGeminiVerbatimMixedParts(t *testing.T) {
	c, _, _ := geminiFixture(t, types.RelayFormatOpenAI, false)
	parts := []dto.GeminiPart{{Text: "hel"}, {Text: "lo"}, {Text: "note", Thought: true}, {Text: "\n"}, {Text: "answer"}}
	frame := &dto.GeminiChatResponse{Candidates: []dto.GeminiChatCandidate{{Content: dto.GeminiChatContent{Parts: parts}}}}
	msg := responseGeminiChat2OpenAI(c, frame).Choices[0].Message
	require.Equal(t, "hello\nanswer", msg.StringContent())
	require.Equal(t, "note", msg.GetReasoningContent())
	chunk := (&geminiChunker{}).chunk(frame)
	require.Equal(t, "hello\nanswer", chunk.Choices[0].Delta.GetContentString())
	require.Equal(t, "note", chunk.Choices[0].Delta.GetReasoningContent())
}

func TestGeminiTerminalDelivery(t *testing.T) {
	for _, reason := range []string{"MALFORMED_RESPONSE", "MALFORMED_FUNCTION_CALL", "STOP", "SAFETY", "MAX_TOKENS"} {
		c, w, info := geminiFixture(t, types.RelayFormatOpenAI, true)
		_, err := GeminiChatStreamHandler(c, info, geminiSSE(`{"candidates":[{"content":{"parts":[]},"finishReason":"`+reason+`"}]}`))
		switch reason {
		case "SAFETY", "MAX_TOKENS":
			require.Nil(t, err, reason)
			require.Contains(t, w.Body.String(), "[DONE]")
		default:
			require.NotNil(t, err, reason)
			require.True(t, types.IsSkipRetryError(err))
			require.Empty(t, w.Body.String())
		}
	}
	for _, tail := range []string{`{"candidates":[{"finishReason":"MALFORMED_RESPONSE"}],"usageMetadata":{"totalTokenCount":5,"promptTokenCount":3,"candidatesTokenCount":2}}`, ""} {
		c, w, info := geminiFixture(t, types.RelayFormatOpenAI, true)
		frames := []string{`{"candidates":[{"content":{"parts":[{"text":"partial"}]}}],"usageMetadata":{"totalTokenCount":5,"promptTokenCount":3,"candidatesTokenCount":2}}`}
		if tail != "" {
			frames = append(frames, tail)
		}
		u, err := GeminiChatStreamHandler(c, info, geminiSSE(frames...))
		require.NotNil(t, err)
		require.True(t, types.IsSkipRetryError(err))
		require.Same(t, err, info.PartialStreamError)
		require.Equal(t, 5, u.TotalTokens)
		require.Contains(t, w.Body.String(), "partial")
		require.NotContains(t, w.Body.String(), "[DONE]")
	}
	c, w, info := geminiFixture(t, types.RelayFormatClaude, true)
	_, err := GeminiChatStreamHandler(c, info, geminiSSE(`{"candidates":[{"content":{"parts":[{"thought":true,"text":"reason"},{"text":"answer"}]},"finishReason":"MAX_TOKENS"}],"usageMetadata":{"totalTokenCount":3,"promptTokenCount":1,"candidatesTokenCount":2}}`))
	require.Nil(t, err)
	body := w.Body.String()
	require.Contains(t, body, `"thinking":"reason"`)
	require.Contains(t, body, `"text":"answer"`)
	require.Contains(t, body, `"stop_reason":"max_tokens"`)
	require.Equal(t, 1, strings.Count(body, "event: message_stop"))
}
