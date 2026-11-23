package gemini

import (
	"testing"

	"github.com/QuantumNous/new-api/dto"
	relaycommon "github.com/QuantumNous/new-api/relay/common"

	"github.com/stretchr/testify/require"
)

func TestGemini4ThinkingLevels(t *testing.T) {
	for _, tc := range []struct{ name, effort, want string }{
		{"gemini-4-hydrogen-preview", "max", "xhigh"}, {"gemini-4-neon-xhigh", "low", "xhigh"}, {"gemini-4-argon", "max", "high"}, {"gemini-4-carbon", "minimal", "low"}, {"gemini-4-argon-thinking-512", "", "low"}, {"gemini-4-argon", "", ""},
	} {
		var req dto.GeminiChatRequest
		info := &relaycommon.RelayInfo{ChannelMeta: &relaycommon.ChannelMeta{UpstreamModelName: tc.name}}
		require.NoError(t, applyGemini4Effort(&req, info, dto.GeneralOpenAIRequest{ReasoningEffort: tc.effort}))
		if tc.want == "" {
			require.Nil(t, req.GenerationConfig.ThinkingConfig)
			continue
		}
		require.Equal(t, tc.want, req.GenerationConfig.ThinkingConfig.ThinkingLevel)
		require.Nil(t, req.GenerationConfig.ThinkingConfig.ThinkingBudget)
	}
	info := &relaycommon.RelayInfo{ChannelMeta: &relaycommon.ChannelMeta{UpstreamModelName: "gemini-4-argon"}}
	require.Error(t, applyGemini4Effort(&dto.GeminiChatRequest{}, info, dto.GeneralOpenAIRequest{ReasoningEffort: "none"}))
}
