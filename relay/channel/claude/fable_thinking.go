package claude

import (
	"fmt"
	"net/http"
	"regexp"
	"strings"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/dto"
	"github.com/QuantumNous/new-api/service/openaicompat"
	"github.com/QuantumNous/new-api/setting/reasoning"
	"github.com/QuantumNous/new-api/types"
)

var fable5Name = regexp.MustCompile(`^claude-fable-5([.-][0-9]+)*$`)

func fableModel(model string) (string, string, bool) {
	base, suffix, _ := reasoning.TrimEffortSuffixWithSuffixes(model, []string{"-nothinking", "-thinking", "-none", "-minimal", "-low", "-medium", "-high", "-xhigh", "-max"})
	return base, suffix, fable5Name.MatchString(base)
}

func fableInvalid(message string) error {
	return types.NewErrorWithStatusCode(fmt.Errorf("Fable: %s", message), types.ErrorCodeInvalidRequest, http.StatusBadRequest, types.ErrOptionWithSkipRetry())
}

// Fable 5 uses adaptive thinking only. Never silently turn thinking on for a
// client that explicitly disabled it, or pass a manual budget to the provider.
func applyFableChatThinking(req dto.GeneralOpenAIRequest, out *dto.ClaudeRequest, model, suffix string) error {
	var native *dto.Thinking
	var enabled *bool
	var router struct {
		Enabled   *bool  `json:"enabled"`
		Effort    string `json:"effort"`
		MaxTokens *int   `json:"max_tokens"`
	}
	if len(req.THINKING) > 0 && common.Unmarshal(req.THINKING, &native) != nil {
		return fableInvalid("invalid thinking object")
	}
	if len(req.EnableThinking) > 0 && common.Unmarshal(req.EnableThinking, &enabled) != nil {
		return fableInvalid("enable_thinking must be boolean")
	}
	if len(req.Reasoning) > 0 && common.Unmarshal(req.Reasoning, &router) != nil {
		return fableInvalid("invalid reasoning object")
	}
	effort := strings.ToLower(strings.TrimSpace(req.ReasoningEffort))
	routerEffort := strings.ToLower(strings.TrimSpace(router.Effort))
	budget := router.MaxTokens
	disabled := suffix == "none" || suffix == "nothinking" || effort == "none" || routerEffort == "none" || (enabled != nil && !*enabled) || (router.Enabled != nil && !*router.Enabled)
	display := ""
	if native != nil {
		display = native.Display
		switch native.Type {
		case "off", "disabled", "none":
			disabled = true
		case "adaptive", "enabled":
		default:
			return fableInvalid("unsupported thinking type")
		}
		if native.BudgetTokens != nil {
			budget = native.BudgetTokens
		}
	}
	if disabled || (budget != nil && *budget == 0) {
		return fableInvalid("this model cannot disable thinking")
	}
	if choice, ok := out.ToolChoice.(*dto.ClaudeToolChoice); ok && (choice.Type == "tool" || choice.Type == "any") {
		return fableInvalid("forced tool use is not supported")
	}
	if effort == "" {
		effort = routerEffort
	}
	if suffix != "" && suffix != "thinking" {
		effort = suffix
	}
	if effort == "" && budget != nil {
		switch {
		case *budget < -1:
			return fableInvalid("invalid reasoning budget")
		case *budget < 0:
			effort = "high"
		case *budget <= 1024:
			effort = "low"
		case *budget <= 8192:
			effort = "medium"
		default:
			effort = "high"
		}
	}
	if effort == "minimal" {
		effort = "low"
	}
	switch effort {
	case "", "low", "medium", "high", "xhigh", "max":
	default:
		return fableInvalid("unsupported reasoning effort")
	}
	out.Model = model
	out.TopP, out.TopK, out.Temperature = nil, nil, nil
	if native != nil || enabled != nil || router.Enabled != nil || suffix == "thinking" || effort != "" {
		out.Thinking = &dto.Thinking{Type: "adaptive", Display: display}
	}
	if effort != "" {
		return openaicompat.SetClaudeOutputEffort(out, effort)
	}
	return nil
}
