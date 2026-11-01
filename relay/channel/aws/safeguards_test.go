package aws

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/dto"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	"github.com/QuantumNous/new-api/setting/model_setting"
	"github.com/QuantumNous/new-api/types"
	"github.com/aws/aws-sdk-go-v2/service/bedrockruntime"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func TestDoAwsClientRequestPreservesSafeguards(t *testing.T) {
	settings := model_setting.GetGlobalSettings()
	originalPassthrough := settings.PassThroughRequestEnabled
	settings.PassThroughRequestEnabled = false
	t.Cleanup(func() { settings.PassThroughRequestEnabled = originalPassthrough })

	const legacyBody = `{"anthropic_version":"bedrock-2023-05-31","anthropic_beta":["dangerous-tool-use-2026-09-03"],"messages":[{"role":"user","content":"hello"}],"max_tokens":512}`
	for _, tt := range []struct{ name, safeguards string }{
		{name: "omitted"},
		{name: "empty array", safeguards: `[]`},
		{name: "dangerous tool use", safeguards: `[{"type":"dangerous_tool_use","classifier_context":{"v":1,"permission_mode":"auto"}}]`},
		// Leave invalid values intact so the upstream rejects them instead of
		// silently disabling the requested safeguard.
		{name: "invalid object", safeguards: `{"enabled":true,"policy":"strict"}`},
		{name: "explicit null", safeguards: `null`},
	} {
		for _, stream := range []bool{false, true} {
			name := tt.name + "/nonstream"
			if stream {
				name = tt.name + "/stream"
			}
			t.Run(name, func(t *testing.T) {
				input := map[string]any{
					"model": "claude-sonnet-5", "max_tokens": 512, "stream": stream,
					"messages": []map[string]any{{"role": "user", "content": "hello"}},
				}
				if tt.safeguards != "" {
					input["safeguards"] = json.RawMessage(tt.safeguards)
				}
				raw, err := common.Marshal(input)
				require.NoError(t, err)
				var request dto.ClaudeRequest
				require.NoError(t, common.Unmarshal(raw, &request))
				ctx, _ := gin.CreateTestContext(httptest.NewRecorder())
				ctx.Request = httptest.NewRequest(http.MethodPost, "/v1/messages", bytes.NewReader(raw))
				ctx.Request.Header.Set("anthropic-beta", "dangerous-tool-use-2026-09-03")
				info := &relaycommon.RelayInfo{
					OriginModelName: "claude-sonnet-5",
					IsStream:        stream,
					RelayFormat:     types.RelayFormatClaude,
					ChannelMeta: &relaycommon.ChannelMeta{
						ApiKey:            "access-key|secret-key|us-east-1",
						UpstreamModelName: "global.anthropic.claude-sonnet-5",
					},
				}
				adaptor := &Adaptor{}
				converted, err := adaptor.ConvertClaudeRequest(ctx, info, &request)
				require.NoError(t, err)
				raw, err = common.Marshal(converted)
				require.NoError(t, err)
				_, err = doAwsClientRequest(ctx, info, adaptor, bytes.NewReader(raw))
				require.NoError(t, err)
				var outbound []byte
				if stream {
					sdkRequest, ok := adaptor.AwsReq.(*bedrockruntime.InvokeModelWithResponseStreamInput)
					require.True(t, ok)
					outbound = sdkRequest.Body
				} else {
					sdkRequest, ok := adaptor.AwsReq.(*bedrockruntime.InvokeModelInput)
					require.True(t, ok)
					outbound = sdkRequest.Body
				}
				if tt.safeguards == "" {
					require.Equal(t, legacyBody, string(outbound))
					return
				}
				var expected map[string]json.RawMessage
				require.NoError(t, common.UnmarshalJsonStr(legacyBody, &expected))
				expected["safeguards"] = json.RawMessage(tt.safeguards)
				want, err := common.Marshal(expected)
				require.NoError(t, err)
				require.JSONEq(t, string(want), string(outbound))
			})
		}
	}
}
