package gemini

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	"github.com/QuantumNous/new-api/dto"
	rc "github.com/QuantumNous/new-api/relay/constant"
	"github.com/QuantumNous/new-api/relay/helper"
	"github.com/QuantumNous/new-api/service"
	"github.com/QuantumNous/new-api/types"

	"github.com/stretchr/testify/require"
	"github.com/tidwall/gjson"
)

func TestGeminiResponsesRoundTrip(t *testing.T) {
	service.InitHttpClient()
	for _, stream := range []bool{false, true} {
		t.Run(fmt.Sprint(stream), func(t *testing.T) {
			c, w, info := geminiFixture(t, types.RelayFormatOpenAIResponses, stream)
			info.RelayMode = rc.RelayModeResponses
			info.ChannelType = constant.ChannelTypeGemini
			info.ApiKey = "fixture"
			var requests atomic.Int32
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				requests.Add(1)
				if stream {
					if !strings.HasSuffix(r.URL.Path, ":streamGenerateContent") || r.URL.Query().Get("alt") != "sse" {
						t.Error("wrong streaming URL")
					}
				} else if !strings.HasSuffix(r.URL.Path, "gemini-4-argon:generateContent") {
					t.Error("wrong URL")
				}
				if r.Method != "POST" || r.Header.Get("x-goog-api-key") != "fixture" {
					t.Error("invalid upstream request")
				}
				b, _ := io.ReadAll(r.Body)
				if !gjson.GetBytes(b, "contents.0.parts.0.text").Exists() || gjson.GetBytes(b, "input").Exists() || gjson.GetBytes(b, "generationConfig.maxOutputTokens").Raw != "0" || gjson.GetBytes(b, "generationConfig.topP").Raw != "0" {
					t.Error("request not converted")
				}
				if gjson.GetBytes(b, "tools.0.functionDeclarations.0.parametersJsonSchema.properties.id.const").Raw != "9007199254740993" {
					t.Error("function schema changed")
				}
				const first = `{"candidates":[{"content":{"parts":[{"text":"a"},{"text":"b"},{"thought":true,"text":"note"}]}}]}`
				const last = `{"candidates":[{"finishReason":"STOP","content":{"parts":[{"functionCall":{"id":"provider_call","name":"lookup","args":{"id":9007199254740993}},"thoughtSignature":"c2ln"}]}}],"usageMetadata":{"promptTokenCount":5,"candidatesTokenCount":2,"thoughtsTokenCount":1,"totalTokenCount":8,"cachedContentTokenCount":2}}`
				if stream {
					w.Header().Set("Content-Type", "text/event-stream")
					fmt.Fprint(w, "data: "+first+"\n\ndata: "+last+"\n\n")
				} else {
					w.Header().Set("Content-Type", "application/json")
					fmt.Fprint(w, strings.Replace(last, `"parts":[`, `"parts":[{"text":"a"},{"text":"b"},{"thought":true,"text":"note"},`, 1))
				}
			}))
			defer server.Close()
			info.ChannelBaseUrl = server.URL
			var req dto.OpenAIResponsesRequest
			require.NoError(t, common.UnmarshalJsonStr(fmt.Sprintf(`{"model":"gemini-4-argon","instructions":"s","input":"hi","stream":%v,"store":false,"top_p":0,"max_output_tokens":0,"tools":[{"type":"function","name":"lookup","parameters":{"type":"object","properties":{"id":{"type":"integer","const":9007199254740993}}}}]}`, stream), &req))
			info.Request = &req
			ad := &Adaptor{}
			wireReq, e := ad.ConvertOpenAIResponsesRequest(c, info, req)
			require.NoError(t, e)
			b, e := common.Marshal(wireReq)
			require.NoError(t, e)
			up, e := ad.DoRequest(c, info, bytes.NewReader(b))
			require.NoError(t, e)
			require.EqualValues(t, 1, requests.Load())
			u, apiErr := ad.DoResponse(c, up.(*http.Response), info)
			require.Nil(t, apiErr)
			require.Equal(t, 8, u.(*dto.Usage).TotalTokens)
			body := w.Body.String()
			var response string
			if stream {
				var events []string
				for _, line := range strings.Split(body, "\n") {
					if !strings.HasPrefix(line, "data: ") {
						continue
					}
					j := gjson.Parse(strings.TrimPrefix(line, "data: "))
					require.Equal(t, int64(len(events)), j.Get("sequence_number").Int())
					events = append(events, j.Get("type").String())
					if j.Get("type").String() == "response.completed" {
						response = j.Get("response").Raw
					}
				}
				require.Equal(t, "response.created", events[0])
				require.Equal(t, "response.completed", events[len(events)-1])
				require.Equal(t, 1, strings.Count(body, "event: response.completed\n"))
				require.Contains(t, events, "response.function_call_arguments.delta")
				require.Contains(t, events, "response.content_part.done")
				require.NotContains(t, body, "[DONE]")
			} else {
				response = body
			}
			require.Equal(t, "completed", gjson.Get(response, "status").String())
			require.Equal(t, "resp_fixture", gjson.Get(response, "id").String())
			require.Equal(t, "ab", gjson.Get(response, `output.#(type=="message").content.0.text`).String())
			require.Equal(t, "note", gjson.Get(response, `output.#(type=="reasoning").summary.0.text`).String())
			item := gjson.Get(response, `output.#(type=="function_call")`)
			require.Equal(t, `{"id":9007199254740993}`, item.Get("arguments").String())
			require.Equal(t, "c2ln", item.Get("extra_content.google.thought_signature").String())
			require.Equal(t, "provider_call", item.Get("call_id").String())
			require.Equal(t, int64(5), gjson.Get(response, "usage.input_tokens").Int())
			require.Equal(t, int64(3), gjson.Get(response, "usage.output_tokens").Int())
			require.Equal(t, int64(2), gjson.Get(response, "usage.input_tokens_details.cached_tokens").Int())
			require.Equal(t, int64(1), gjson.Get(response, "usage.output_tokens_details.reasoning_tokens").Int())
			require.Equal(t, "9007199254740993", gjson.Get(response, "tools.0.parameters.properties.id.const").Raw)
			// Real clients append the complete returned output, not just function calls.
			req.Input = []byte(strings.TrimSuffix(gjson.Get(response, "output").Raw, "]") + `,{"type":"function_call_output","call_id":"provider_call","output":"{\"id\":9007199254740993}"}]`)
			next, e := ad.ConvertOpenAIResponsesRequest(c, info, req)
			require.NoError(t, e)
			nb, e := common.Marshal(next)
			require.NoError(t, e)
			p := gjson.GetBytes(nb, `contents.0.parts.#(functionCall.name=="lookup")`)
			require.Equal(t, "c2ln", p.Get("thoughtSignature").String())
			require.Equal(t, "note", gjson.GetBytes(nb, `contents.0.parts.#(thought==true).text`).String())
			require.Equal(t, "9007199254740993", p.Get("functionCall.args.id").Raw)
			require.Equal(t, "lookup", gjson.GetBytes(nb, "contents.1.parts.0.functionResponse.name").String())
			require.Equal(t, "9007199254740993", gjson.GetBytes(nb, "contents.1.parts.0.functionResponse.response.id").Raw)
			require.EqualValues(t, 1, requests.Load())
		})
	}
}

func TestGeminiResponsesRejectsUnsupportedSemantics(t *testing.T) {
	for _, extra := range []string{`"previous_response_id":"prev"`, `"store":true`, `"background":true`, `"tools":[{"type":"web_search"}]`, `"tools":[{"type":"custom","name":"f"}]`, `"include":["reasoning.encrypted_content"]`, `"truncation":"auto"`, `"parallel_tool_calls":false`} {
		c, _, info := geminiFixture(t, types.RelayFormatOpenAIResponses, false)
		c.Request = httptest.NewRequest("POST", "/v1/responses", strings.NewReader(`{"model":"gemini-4-argon","input":"hello",`+extra+`}`))
		c.Request.Header.Set("Content-Type", "application/json")
		req, e := helper.GetAndValidateResponsesRequest(c)
		require.NoError(t, e)
		_, e = (&Adaptor{}).ConvertOpenAIResponsesRequest(c, info, *req)
		var routed *types.NewAPIError
		require.ErrorAs(t, e, &routed, extra)
		require.Equal(t, 400, routed.StatusCode)
		require.True(t, types.IsSkipRetryError(routed))
	}
}

func TestGeminiResponsesTerminals(t *testing.T) {
	const partial = `{"candidates":[{"content":{"parts":[{"text":"partial"}]}}],"usageMetadata":{"totalTokenCount":3,"promptTokenCount":1,"candidatesTokenCount":2}}`
	for _, tc := range []struct{ reason, want string }{{"MAX_TOKENS", "max_output_tokens"}, {"SAFETY", "content_filter"}, {"MALFORMED_RESPONSE", ""}, {"", ""}} {
		c, w, info := geminiFixture(t, types.RelayFormatOpenAIResponses, true)
		info.RelayMode = rc.RelayModeResponses
		frames := []string{partial}
		if tc.reason != "" {
			frames = append(frames, `{"candidates":[{"finishReason":"`+tc.reason+`"}]}`)
		}
		u, err := geminiResponsesStreamHandler(c, info, geminiSSE(frames...))
		require.Equal(t, 3, u.TotalTokens)
		body := w.Body.String()
		require.Contains(t, body, "partial")
		require.NotContains(t, body, "event: response.completed")
		if tc.want != "" {
			require.Nil(t, err)
			require.Contains(t, body, "event: response.incomplete")
			require.Contains(t, body, tc.want)
			require.True(t, info.IsResponsesIncomplete())
		} else {
			require.NotNil(t, err)
			require.Same(t, err, info.PartialStreamError)
			require.True(t, types.IsSkipRetryError(err))
			require.NoError(t, helper.SendInBandStreamError(c, types.RelayFormatOpenAIResponses, err))
			require.Equal(t, 1, strings.Count(w.Body.String(), "event: response.failed"))
		}
	}
}
