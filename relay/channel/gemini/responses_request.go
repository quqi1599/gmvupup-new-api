package gemini

import (
	"bytes"
	"encoding/json"
	"errors"
	"net/http"
	"strings"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	"github.com/QuantumNous/new-api/dto"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	"github.com/QuantumNous/new-api/service/openaicompat"
	"github.com/QuantumNous/new-api/types"

	"github.com/gin-gonic/gin"
)

func unsupportedResponse(feature string) *types.NewAPIError {
	return types.NewErrorWithStatusCode(errors.New("Gemini Responses cannot preserve "+feature), types.ErrorCodeRequestFeatureUnsupported, http.StatusBadRequest, types.ErrOptionWithSkipRetry())
}

func rawSet(raw json.RawMessage) bool {
	v := bytes.TrimSpace(raw)
	return len(v) > 0 && !bytes.Equal(v, []byte("null"))
}

type responseFunction struct {
	Name        string          `json:"name"`
	Description string          `json:"description,omitempty"`
	Parameters  json.RawMessage `json:"parametersJsonSchema,omitempty"`
}

func convertGeminiResponsesRequest(c *gin.Context, info *relaycommon.RelayInfo, req dto.OpenAIResponsesRequest) (*dto.GeminiChatRequest, error) {
	if info == nil || info.ChannelMeta == nil {
		return nil, errors.New("missing Gemini channel metadata")
	}
	if c != nil && common.GetContextKeyBool(c, constant.ContextKeyResponsesBackground) {
		return nil, unsupportedResponse("background jobs")
	}
	if rawSet(req.Conversation) || req.PreviousResponseID != "" || rawSet(req.ContextManagement) || rawSet(req.Prompt) {
		return nil, unsupportedResponse("server-side conversation or prompt state")
	}
	if rawSet(req.Store) && string(bytes.TrimSpace(req.Store)) != "false" {
		return nil, unsupportedResponse("stored responses")
	}
	if rawSet(req.Include) {
		var fields []string
		if err := common.Unmarshal(req.Include, &fields); err != nil || len(fields) > 0 {
			return nil, unsupportedResponse("requested include fields")
		}
	}
	if rawSet(req.Truncation) && common.JsonRawMessageToString(req.Truncation) != "disabled" {
		return nil, unsupportedResponse("automatic truncation")
	}
	if req.TopLogProbs != nil || req.MaxToolCalls != nil || rawSet(req.PromptCacheKey) || rawSet(req.PromptCacheRetention) || rawSet(req.Preset) {
		return nil, unsupportedResponse("log probabilities, tool limits, presets or prompt cache controls")
	}
	if req.ServiceTier != "" && req.ServiceTier != "auto" && req.ServiceTier != "default" || rawSet(req.SafetyIdentifier) || rawSet(req.User) {
		return nil, unsupportedResponse("provider-specific service tier or user identifiers")
	}
	if req.StreamOptions != nil && req.StreamOptions.IncludeObfuscation {
		return nil, unsupportedResponse("stream obfuscation")
	}
	if rawSet(req.ParallelToolCalls) && string(bytes.TrimSpace(req.ParallelToolCalls)) != "true" {
		return nil, unsupportedResponse("disabling parallel tool calls")
	}

	nativeTools, names, err := geminiResponseTools(req.Tools)
	if err != nil {
		return nil, err
	}
	extras, err := geminiResponseHistory(req.Input)
	if err != nil {
		return nil, err
	}
	// Tools are rendered directly as original JSON Schemas. Do not route a
	// user function named googleSearch or codeExecution to a hosted tool.
	req.Tools = nil
	chat, err := openaicompat.ResponsesToGeminiPivot(&req)
	if err != nil {
		var routed *types.NewAPIError
		if errors.As(err, &routed) {
			return nil, routed
		}
		return nil, geminiInvalid("invalid or unsupported Responses input")
	}
	if len(chat.Verbosity) > 0 {
		return nil, unsupportedResponse("output verbosity")
	}
	if req.Reasoning != nil && req.Reasoning.Summary != "" && req.Reasoning.Summary != "auto" {
		return nil, unsupportedResponse("requested reasoning summary format")
	}
	var choice *dto.ToolConfig
	switch v := chat.ToolChoice.(type) {
	case nil:
	case string:
		if v != "auto" && v != "none" && v != "required" {
			return nil, unsupportedResponse("tool choice")
		}
		if v == "required" && len(names) == 0 {
			return nil, geminiInvalid("required tool choice needs functions")
		}
		if len(names) > 0 {
			choice = convertToolChoiceToGeminiConfig(v)
		}
	case map[string]any:
		fn, _ := v["function"].(map[string]any)
		name, _ := fn["name"].(string)
		if v["type"] != "function" || !names[name] {
			return nil, unsupportedResponse("unknown or non-function tool choice")
		}
		choice = convertToolChoiceToGeminiConfig(v)
	default:
		return nil, unsupportedResponse("tool choice")
	}
	for i := range chat.Messages {
		calls := chat.Messages[i].ParseToolCalls()
		if len(calls) == 0 {
			continue
		}
		for j := range calls {
			calls[j].ExtraContent = extras[calls[j].ID]
		}
		chat.Messages[i].SetToolCalls(calls)
	}
	var schema json.RawMessage
	if f := chat.ResponseFormat; f != nil && f.Type == "json_schema" {
		var def struct {
			Schema json.RawMessage `json:"schema"`
		}
		if common.Unmarshal(f.JsonSchema, &def) != nil || common.GetJsonType(def.Schema) != "object" {
			return nil, geminiInvalid("invalid JSON output schema")
		}
		schema = def.Schema
		chat.ResponseFormat = nil
	}
	request, err := convertChatToGemini(c, *chat, info, true)
	if err != nil {
		return nil, err
	}
	if req.Reasoning != nil && req.Reasoning.Effort != "" && request.GenerationConfig.ThinkingConfig == nil {
		return nil, unsupportedResponse("reasoning effort on this model and channel")
	}
	if len(nativeTools) > 0 {
		request.SetTools([]dto.GeminiChatTool{{FunctionDeclarations: nativeTools}})
	}
	request.ToolConfig = choice
	if schema != nil {
		request.GenerationConfig.ResponseMimeType = "application/json"
		request.GenerationConfig.ResponseJsonSchema = schema
	}
	return request, nil
}

func geminiResponseTools(raw json.RawMessage) ([]responseFunction, map[string]bool, error) {
	var tools []struct {
		Type, Name, Description string
		Parameters              json.RawMessage
		Strict                  *bool
	}
	if rawSet(raw) && common.Unmarshal(raw, &tools) != nil {
		return nil, nil, geminiInvalid("invalid Responses tools")
	}
	var out []responseFunction
	names := map[string]bool{}
	for _, t := range tools {
		if t.Type != "function" || (t.Strict != nil && *t.Strict) {
			return nil, nil, unsupportedResponse("hosted, custom or strict function tools")
		}
		if strings.TrimSpace(t.Name) == "" || names[t.Name] {
			return nil, nil, geminiInvalid("missing or duplicate function name")
		}
		if rawSet(t.Parameters) && common.GetJsonType(t.Parameters) != "object" {
			return nil, nil, geminiInvalid("invalid function parameters")
		}
		names[t.Name] = true
		out = append(out, responseFunction{t.Name, t.Description, t.Parameters})
	}
	return out, names, nil
}

// Reject input types that the generic Chat pivot cannot represent instead
// of dropping tool definitions, opaque reasoning, item references or media.
func geminiResponseHistory(raw json.RawMessage) (map[string]*dto.GoogleToolExtra, error) {
	extras := map[string]*dto.GoogleToolExtra{}
	if common.GetJsonType(raw) == "string" {
		return extras, nil
	}
	var items []struct {
		Type, Role, ID  string
		CallID          string `json:"call_id"`
		Content, Output json.RawMessage
		Extra           *dto.GoogleToolExtra `json:"extra_content"`
	}
	if common.GetJsonType(raw) != "array" || common.Unmarshal(raw, &items) != nil {
		return nil, geminiInvalid("input must be text or an array of items")
	}
	answered := map[string]bool{}
	for _, item := range items {
		switch item.Type {
		case "", "message":
			switch item.Role {
			case "user", "assistant", "system", "developer":
			default:
				return nil, geminiInvalid("unsupported message role")
			}
			if common.GetJsonType(item.Content) == "string" {
				continue
			}
			var parts []struct {
				Type     string
				ImageURL json.RawMessage `json:"image_url"`
				FileData string          `json:"file_data"`
			}
			if common.GetJsonType(item.Content) != "array" || common.Unmarshal(item.Content, &parts) != nil {
				return nil, geminiInvalid("invalid message content")
			}
			for _, p := range parts {
				switch p.Type {
				case "text", "input_text", "output_text":
				case "input_image":
					if !rawSet(p.ImageURL) {
						return nil, unsupportedResponse("file_id image")
					}
					if item.Role == "system" || item.Role == "developer" {
						return nil, unsupportedResponse("media system instructions")
					}
				case "input_file":
					if p.FileData == "" || item.Role == "system" || item.Role == "developer" {
						return nil, unsupportedResponse("unresolved files or system media")
					}
				default:
					return nil, unsupportedResponse("this message content type")
				}
			}
		case "reasoning":
			// The Gemini-specific pivot validates summary_text and rejects encrypted state.
		case "function_call":
			id := item.CallID
			if id == "" {
				id = item.ID
			}
			if _, dup := extras[id]; id == "" || dup {
				return nil, geminiInvalid("missing or duplicate function call ID")
			}
			if item.Extra != nil && common.GetJsonType(item.Extra.Google.ThoughtSignature) != "string" {
				return nil, geminiInvalid("invalid Google tool signature")
			}
			extras[id] = item.Extra
		case "function_call_output":
			if _, exists := extras[item.CallID]; !exists || answered[item.CallID] {
				return nil, geminiInvalid("function output must match one preceding call")
			}
			answered[item.CallID] = true
			// Responses function outputs are strings or typed content items.
			if common.GetJsonType(item.Output) != "string" && common.GetJsonType(item.Output) != "array" {
				return nil, unsupportedResponse("non-string or untyped function output")
			}
		default:
			return nil, unsupportedResponse("this input item type")
		}
	}
	return extras, nil
}
