package gemini

import (
	"errors"
	"net/http"
	"strconv"
	"strings"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/dto"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	"github.com/QuantumNous/new-api/setting/reasoning"
	"github.com/QuantumNous/new-api/types"
)

func geminiInvalid(reason string) *types.NewAPIError {
	return types.NewErrorWithStatusCode(errors.New(reason), types.ErrorCodeInvalidRequest, http.StatusBadRequest, types.ErrOptionWithSkipRetry())
}

// Mirrors upstream #8412: the Neon/Hydrogen families accept xhigh. Other
// Gemini 4 families cap at high; these are levels, never token budgets.
func applyGemini4Effort(req *dto.GeminiChatRequest, info *relaycommon.RelayInfo, chat ...dto.GeneralOpenAIRequest) error {
	model, suffix, _ := reasoning.TrimEffortSuffix(info.UpstreamModelName)
	effort := ""
	off := strings.HasSuffix(model, "-none") || strings.HasSuffix(model, "-nothinking")
	if len(chat) > 0 {
		effort = strings.ToLower(strings.TrimSpace(chat[0].ReasoningEffort))
		var enabled *bool
		if len(chat[0].EnableThinking) > 0 && common.Unmarshal(chat[0].EnableThinking, &enabled) != nil {
			return geminiInvalid("invalid enable_thinking")
		}
		var control *dto.Thinking
		if len(chat[0].THINKING) > 0 && common.Unmarshal(chat[0].THINKING, &control) != nil {
			return geminiInvalid("invalid thinking object")
		}
		off = off || effort == "none" || (enabled != nil && !*enabled) || (control != nil && (control.Type == "disabled" || control.Type == "none" || control.Type == "off"))
	}
	if off {
		return geminiInvalid("Gemini 4 does not support disabling thinking")
	}
	if suffix != "" {
		effort = suffix
	}
	if _, b, ok := strings.Cut(model, "-thinking-"); ok {
		n, e := strconv.Atoi(b)
		if e != nil || n == 0 || n < -1 {
			return geminiInvalid("invalid Gemini 4 thinking budget")
		}
		switch {
		case n < 0:
			effort = "high"
		case n <= 1024:
			effort = "low"
		case n <= 8192:
			effort = "medium"
		default:
			effort = "high"
		}
	}
	if strings.HasSuffix(model, "-thinking") && effort == "" {
		effort = "high"
	}
	switch effort {
	case "":
		return nil
	case "minimal":
		effort = "low"
	case "low", "medium", "high":
	case "xhigh", "max":
		effort = "high"
		if strings.HasPrefix(model, "gemini-4-hydrogen") || strings.HasPrefix(model, "gemini-4-neon") {
			effort = "xhigh"
		}
	default:
		return geminiInvalid("unsupported Gemini 4 reasoning effort")
	}
	req.GenerationConfig.ThinkingConfig = &dto.GeminiThinkingConfig{IncludeThoughts: true, ThinkingLevel: effort}
	info.ReasoningEffort = effort
	return nil
}
