package gemini

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/dto"
	"github.com/QuantumNous/new-api/types"
)

type geminiOutcome struct {
	visible map[int64]bool
	ended   map[int64]bool
}

func geminiBadResult(message string) *types.NewAPIError {
	// A provider response already exists; retrying could duplicate accepted work.
	return types.NewErrorWithStatusCode(errors.New(message), types.ErrorCodeUpstreamStreamIncomplete, http.StatusBadGateway, types.ErrOptionWithSkipRetry())
}

func geminiPolicyFinish(reason string) bool {
	switch reason {
	case "SAFETY", "RECITATION", "BLOCKLIST", "PROHIBITED_CONTENT", "SPII", "LANGUAGE", "MODEL_ARMOR", "IMAGE_SAFETY", "IMAGE_PROHIBITED_CONTENT", "IMAGE_RECITATION":
		return true
	}
	return false
}

// Validate before Accept or downstream writes. MAX_TOKENS and policy stops
// are valid incomplete outcomes even without text; thoughts alone are not a
// completed answer. Inspect metadata-only frames without committing headers.
func (s *geminiOutcome) observe(frame *dto.GeminiChatResponse, native bool) (deliver bool, fail *types.NewAPIError) {
	if s.visible == nil {
		s.visible = map[int64]bool{}
		s.ended = map[int64]bool{}
	}
	for _, c := range frame.Candidates {
		if _, ok := s.visible[c.Index]; !ok {
			s.visible[c.Index] = false
		}
		for _, p := range c.Content.Parts {
			if p.FunctionCall != nil {
				raw, fromWire := p.FunctionCall.Arguments.(json.RawMessage)
				if p.FunctionCall.FunctionName == "" || (fromWire && common.GetJsonType(raw) != "object") {
					return false, geminiBadResult("Gemini returned an invalid function call")
				}
			}
			usable := (!p.Thought && p.Text != "") || (p.FunctionCall != nil && p.FunctionCall.FunctionName != "") || (p.InlineData != nil && p.InlineData.Data != "") || p.ExecutableCode != nil || p.CodeExecutionResult != nil
			if usable {
				s.visible[c.Index] = true
			}
			deliver = deliver || usable || p.Thought && p.Text != "" || native && len(p.ThoughtSignature) > 0
		}
		if native && len(c.GroundingMetadata) > 0 {
			var meta struct {
				Web       []string `json:"webSearchQueries"`
				Image     []string `json:"imageSearchQueries"`
				Retrieval []string `json:"retrievalQueries"`
				Chunks    []any    `json:"groundingChunks"`
			}
			if common.Unmarshal(c.GroundingMetadata, &meta) == nil && len(meta.Web)+len(meta.Image)+len(meta.Retrieval)+len(meta.Chunks) > 0 {
				s.visible[c.Index] = true
				deliver = true
			}
		}
		if c.FinishReason == nil || *c.FinishReason == "" || *c.FinishReason == "FINISH_REASON_UNSPECIFIED" {
			continue
		}
		reason := *c.FinishReason
		switch {
		case reason == "STOP":
			if !s.visible[c.Index] {
				return false, geminiBadResult("Gemini stopped without visible output")
			}
		case reason == "MAX_TOKENS", reason == "CONTINUATION", geminiPolicyFinish(reason):
		case reason == "MALFORMED_RESPONSE" || reason == "MALFORMED_FUNCTION_CALL" || reason == "UNEXPECTED_TOOL_CALL" || reason == "TOO_MANY_TOOL_CALLS":
			return false, geminiBadResult("Gemini returned " + reason)
		default:
			return false, geminiBadResult("Gemini returned an abnormal finish reason")
		}
		s.ended[c.Index] = true
		deliver = true
	}
	return deliver, nil
}
