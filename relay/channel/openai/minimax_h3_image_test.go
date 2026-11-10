package openai

import (
	"fmt"
	"strings"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/dto"
	"github.com/QuantumNous/new-api/types"
	"github.com/stretchr/testify/require"
	"github.com/tidwall/gjson"
)

const h3Usage = `{"generated_images":1,"input_tokens":10,"output_tokens":20,"total_tokens":30,"input_tokens_details":{"text_tokens":7,"image_tokens":3},"output_tokens_details":{"image_tokens":20}}`

func h3Event(name, fields string) string {
	return "event: " + name + "\ndata: {\"type\":\"" + name + "\"," + fields + "}\n\n"
}

func TestH3ImageNativeResults(t *testing.T) {
	for _, edit := range []bool{false, true} {
		prefix := "image_generation."
		if edit {
			prefix = "image_edit."
		}
		success := h3Event(prefix+"partial_succeeded", `"image_index":1,"url":"https://example.com/final.png","size":"1024x1024","output_format":"png","revised_prompt":"result"`)
		failed := h3Event(prefix+"partial_failed", `"image_index":0,"code":"1027","message":"blocked"`)
		completed := h3Event(prefix+"completed", `"created":123,"request_id":"synthetic","usage":`+h3Usage)
		for _, stream := range []bool{false, true} {
			for _, duplicate := range []bool{false, true} {
				t.Run(fmt.Sprintf("edit=%t/stream=%t/duplicate=%t", edit, stream, duplicate), func(t *testing.T) {
					body := ": keepalive\n\n" + success + failed
					if duplicate {
						body += success
					}
					body += completed
					c, w, resp, info := imageCompatContext(t, body, "text/event-stream", common.GetPointer(stream), edit)
					usage, err := MiniMaxH3ImageHandler(c, info, resp)
					require.Nil(t, err)
					require.Equal(t, 10, usage.PromptTokens)
					require.Equal(t, 20, usage.CompletionTokens)
					require.Equal(t, 30, usage.TotalTokens)
					require.Equal(t, 3, usage.PromptTokensDetails.ImageTokens)
					require.Equal(t, 20, usage.CompletionTokenDetails.ImageTokens)
					count, known := ConfirmedImageCount(c)
					require.True(t, known)
					require.Equal(t, 1, count)
					require.True(t, ImageUsageReported(c))
					if stream {
						require.Equal(t, 1, strings.Count(w.Body.String(), "event: "+prefix+"partial_succeeded"))
						require.Contains(t, w.Body.String(), "event: "+prefix+"partial_failed")
						require.Contains(t, w.Body.String(), "event: "+prefix+"completed")
					} else {
						require.Equal(t, "1027", gjson.Get(w.Body.String(), "data.0.error.code").String())
						require.Equal(t, "https://example.com/final.png", gjson.Get(w.Body.String(), "data.1.url").String())
						require.Equal(t, "synthetic", gjson.Get(w.Body.String(), "request_id").String())
					}
				})
			}
		}
	}
	for _, format := range []string{"url", "b64_json"} {
		body := `{"created":123,"data":[{"` + format + `":"result","size":"1024x1024"},{"error":{"code":"1027","message":"blocked"}}],"usage":` + h3Usage + `,"request_id":"synthetic","output_format":"png"}`
		c, w, resp, info := imageCompatContext(t, body, "application/json", nil, false)
		usage, err := MiniMaxH3ImageHandler(c, info, resp)
		require.Nil(t, err)
		require.Equal(t, 30, usage.TotalTokens)
		require.JSONEq(t, body, w.Body.String())
		count, known := ConfirmedImageCount(c)
		require.True(t, known)
		require.Equal(t, 1, count)
	}
}

func TestH3ImageIncompleteStreamNeverReportsSuccess(t *testing.T) {
	success := h3Event("image_generation.partial_succeeded", `"image_index":0,"url":"https://example.com/final.png"`)
	for _, suffix := range []string{"", "data: [DONE]\n\n", "data: broken\n\n", h3Event("image_generation.completed", `"usage":`+strings.Replace(h3Usage, `"generated_images":1`, `"generated_images":2`, 1))} {
		for _, stream := range []bool{false, true} {
			c, w, resp, info := imageCompatContext(t, success+suffix, "text/event-stream", common.GetPointer(stream), false)
			usage, err := MiniMaxH3ImageHandler(c, info, resp)
			require.NotNil(t, err)
			require.NotNil(t, usage)
			require.True(t, types.IsSkipRetryError(err))
			require.Same(t, err, info.PartialStreamError)
			count, known := ConfirmedImageCount(c)
			require.True(t, known)
			require.Equal(t, 1, count)
			require.NotContains(t, w.Body.String(), "image_generation.completed")
		}
	}
}

func TestH3ImageAllFailedStream(t *testing.T) {
	body := h3Event("image_generation.partial_failed", `"image_index":0,"code":"1027","message":"blocked"`) +
		h3Event("error", `"error":{"type":"server_error","message":"blocked","http_code":"422"},"usage":{"generated_images":0,"input_tokens":0,"output_tokens":0,"total_tokens":0},"request_id":"synthetic"`)
	c, w, resp, info := imageCompatContext(t, body, "text/event-stream", common.GetPointer(true), false)
	usage, err := MiniMaxH3ImageHandler(c, info, resp)
	require.NotNil(t, err)
	require.True(t, types.IsSkipRetryError(err))
	require.Equal(t, &dto.Usage{}, usage)
	count, known := ConfirmedImageCount(c)
	require.True(t, known)
	require.Zero(t, count)
	require.Contains(t, w.Body.String(), "event: error")
	require.NotContains(t, w.Body.String(), ".completed")
}
