package openai

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	relaycommon "github.com/QuantumNous/new-api/relay/common"
	relayconstant "github.com/QuantumNous/new-api/relay/constant"
	"github.com/QuantumNous/new-api/types"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

type chatChunkRecorder struct {
	*httptest.ResponseRecorder
	delivered chan struct{}
	marker    string
}

func (w *chatChunkRecorder) Flush() {
	w.ResponseRecorder.Flush()
	if strings.Contains(w.Body.String(), w.marker) {
		select {
		case <-w.delivered:
		default:
			close(w.delivered)
		}
	}
}

func TestChatDeliversGeneratedChunkBeforeNextUpstreamFrame(t *testing.T) {
	for _, delta := range []string{`{"content":"piece"}`, `{"reasoning_content":"piece"}`, `{"tool_calls":[{"index":0,"function":{"arguments":"piece"}}]}`} {
		t.Run(delta, func(t *testing.T) {
			chunk := `{"choices":[{"index":0,"delta":` + delta + `}]}`
			w := &chatChunkRecorder{httptest.NewRecorder(), make(chan struct{}), "piece"}
			c, _ := gin.CreateTestContext(w)
			c.Request = httptest.NewRequest(http.MethodPost, "/v1/chat/completions", nil)
			r, up := io.Pipe()
			gated := make(chan bool, 1)
			go func() {
				defer up.Close()
				_, _ = io.WriteString(up, "data: "+chunk+"\n\n")
				select {
				case <-w.delivered:
					gated <- true
				case <-time.After(2 * time.Second):
					gated <- false
				}
				_, _ = io.WriteString(up, "data: {\"choices\":[],\"usage\":{\"prompt_tokens\":5,\"completion_tokens\":2}}\n\ndata: [DONE]\n\n")
			}()
			info := &relaycommon.RelayInfo{IsStream: true, RelayMode: relayconstant.RelayModeChatCompletions, RelayFormat: types.RelayFormatOpenAI, ChannelMeta: &relaycommon.ChannelMeta{UpstreamModelName: "deepseek-v4.1-flash"}}
			usage, err := OaiStreamHandler(c, info, &http.Response{StatusCode: 200, Body: r})
			require.Nil(t, err)
			require.True(t, <-gated, "client received the chunk only after upstream was allowed to continue")
			require.Equal(t, 1, strings.Count(w.Body.String(), chunk))
			require.NotContains(t, w.Body.String(), "prompt_tokens")
			require.Equal(t, 1, strings.Count(w.Body.String(), "[DONE]"))
			require.Equal(t, 2, usage.CompletionTokens)
		})
	}
}
