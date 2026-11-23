package claude

import (
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/dto"
	"github.com/QuantumNous/new-api/types"

	"github.com/stretchr/testify/require"
	"github.com/tidwall/gjson"
)

func TestFableAdaptiveThinking(t *testing.T) {
	for _, tc := range []struct {
		model, fields, effort string
		rejected              bool
	}{
		{"claude-fable-5-2", `"reasoning_effort":"high"`, "high", false},
		{"claude-fable-5-2-xhigh", `"reasoning_effort":"low"`, "xhigh", false},
		{"claude-fable-5-2", `"thinking":{"type":"enabled","budget_tokens":1024}`, "low", false},
		{"claude-fable-5-2", `"reasoning_effort":"none"`, "", true},
		{"claude-fable-5-2", `"thinking":{"type":"disabled"},"reasoning_effort":"high"`, "", true},
		{"claude-fable-5-2", `"reasoning":{"enabled":false}`, "", true},
		{"claude-fable-5-2", `"tool_choice":"required"`, "", true},
	} {
		var req dto.GeneralOpenAIRequest
		require.NoError(t, common.UnmarshalJsonStr(`{"model":"`+tc.model+`","messages":[{"role":"user","content":"hi"}],"temperature":0,"top_p":0.5,`+tc.fields+`,"response_format":{"type":"json_schema","json_schema":{"name":"result","schema":{"type":"object","properties":{}}}}}`, &req))
		got, err := RequestOpenAI2ClaudeMessage(nil, req)
		if tc.rejected {
			var e *types.NewAPIError
			require.ErrorAs(t, err, &e)
			require.Equal(t, 400, e.StatusCode)
			require.True(t, types.IsSkipRetryError(e))
			continue
		}
		require.NoError(t, err)
		require.Equal(t, "claude-fable-5-2", got.Model)
		require.Equal(t, "adaptive", got.Thinking.Type)
		require.Nil(t, got.Thinking.BudgetTokens)
		require.Nil(t, got.Temperature)
		require.Nil(t, got.TopP)
		require.Equal(t, tc.effort, gjson.GetBytes(got.OutputConfig, "effort").String())
		require.True(t, gjson.GetBytes(got.OutputConfig, "format.schema").Exists())
	}
	for _, name := range []string{"claude-fable-50", "claude-fable-5-custom", "another-model"} {
		_, _, ok := fableModel(name)
		require.False(t, ok, name)
	}
	var responseReq dto.OpenAIResponsesRequest
	require.NoError(t, common.UnmarshalJsonStr(`{"model":"claude-fable-5-2","input":"hi","reasoning":{"effort":"max"}}`, &responseReq))
	got, err := (&Adaptor{}).ConvertOpenAIResponsesRequest(nil, nil, responseReq)
	require.NoError(t, err)
	require.Equal(t, "max", gjson.GetBytes(got.(*dto.ClaudeRequest).OutputConfig, "effort").String())
}
