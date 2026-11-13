package dto

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"errors"
	"github.com/gin-gonic/gin"
	"image"
	_ "image/jpeg"
	_ "image/png"
	"strings"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/types"
)

// DecisionsRequest is the native decisions protocol used by TypeSafe. State is shared
// across questions; it must not be expanded into one chat request per question.
type DecisionsRequest struct {
	Model     string                       `json:"model"`
	State     json.RawMessage              `json:"state"`
	Questions map[string]DecisionsQuestion `json:"questions"`
	Assets    map[string]DecisionsAsset    `json:"assets,omitempty"`
	Stream    *bool                        `json:"stream,omitempty"`
}

type DecisionsAsset struct {
	Type       string  `json:"type"`
	Format     *string `json:"format,omitempty"`
	DataBase64 *string `json:"data_base64,omitempty"`
	Part       *string `json:"part,omitempty"`
}

// UnmarshalJSON rejects unknown top-level fields rather than dropping them.
func (r *DecisionsRequest) UnmarshalJSON(data []byte) error {
	type plain DecisionsRequest
	var fields map[string]json.RawMessage
	if err := common.Unmarshal(data, &fields); err != nil {
		return err
	}
	for k := range fields {
		switch k {
		case "model", "state", "questions", "assets", "stream":
		default:
			return errors.New("unsupported decisions request field: " + k)
		}
	}
	return common.DecodeJson(bytes.NewReader(data), (*plain)(r))
}

type DecisionsQuestion struct {
	Type         string          `json:"type"`
	Instructions json.RawMessage `json:"instructions"`
	Criteria     json.RawMessage `json:"criteria,omitempty"`
}

// IsStream always returns false; Validate rejects an explicit streaming request.
func (r *DecisionsRequest) IsStream(*gin.Context) bool { return false }

// SetModelName applies the channel model mapping to the upstream request.
func (r *DecisionsRequest) SetModelName(model string) { r.Model = model }

// GetTokenCountMeta supplies shared state and questions for the gateway token estimate.
// Actual settlement uses the token counts returned by the provider.
func (r *DecisionsRequest) GetTokenCountMeta() *types.TokenCountMeta {
	questions, _ := common.Marshal(r.Questions)
	return &types.TokenCountMeta{
		TokenType:   types.TokenTypeTokenizer,
		CombineText: string(r.State) + "\n" + string(questions),
	}
}

// Validate checks required fields and the criteria shape for noul, choice and score
// questions, and rejects streaming before any upstream call.
func (r *DecisionsRequest) Validate() error {
	if r == nil || strings.TrimSpace(r.Model) == "" {
		return errors.New("model is required")
	}
	if r.Stream != nil && *r.Stream {
		return errors.New("decisions do not support streaming")
	}
	switch common.GetJsonType(r.State) {
	case "string", "object", "array":
	default:
		return errors.New("state must be a string, object or array")
	}
	if len(r.Questions) == 0 {
		return errors.New("questions must not be empty")
	}
	for _, asset := range r.Assets {
		if asset.Type != "image" || asset.Part != nil || asset.DataBase64 == nil || asset.Format == nil || *asset.DataBase64 == "" {
			return errors.New("image asset requires inline image data")
		}
		data, decodeErr := base64.StdEncoding.DecodeString(*asset.DataBase64)
		if decodeErr != nil || len(data) > 20<<20 {
			return errors.New("invalid or oversized base64 image")
		}
		cfg, format, imageErr := image.DecodeConfig(bytes.NewReader(data))
		if imageErr != nil || cfg.Width < 1 || cfg.Height < 1 || (format != "png" && format != "jpeg") {
			return errors.New("invalid PNG or JPEG image")
		}
		if *asset.Format != "png" && *asset.Format != "jpg" && *asset.Format != "jpeg" {
			return errors.New("image format must be png or jpeg")
		}
	}
	for _, q := range r.Questions {
		switch common.GetJsonType(q.Instructions) {
		case "string", "object", "array", "null", "unknown":
		default:
			return errors.New("question instructions must be a string, object or array")
		}
		switch q.Type {
		case "noul":
			if len(q.Criteria) == 0 || string(q.Criteria) == "null" {
				continue
			}
			var criteria map[string]json.RawMessage
			if common.Unmarshal(q.Criteria, &criteria) != nil || criteria == nil {
				return errors.New("noul criteria must be an object")
			}
			for key, value := range criteria {
				if key != "true" && key != "false" {
					continue
				}
				if !validDecisionDescription(value, true) {
					return errors.New("noul criteria descriptions must be strings, objects or arrays")
				}
			}
		case "choice":
			var criteria map[string]json.RawMessage
			if common.Unmarshal(q.Criteria, &criteria) != nil || len(criteria) == 0 || len(criteria) > 255 {
				return errors.New("choice criteria must contain 1 to 255 options")
			}
			for _, value := range criteria {
				if !validDecisionDescription(value, true) {
					return errors.New("invalid choice description")
				}
			}
		case "score":
			var criteria []json.RawMessage
			if common.Unmarshal(q.Criteria, &criteria) != nil || len(criteria) < 1 || len(criteria) > 10 {
				return errors.New("score criteria must contain 1 to 10 levels")
			}
			for _, value := range criteria {
				if !validDecisionDescription(value, false) {
					return errors.New("invalid score level")
				}
			}
		case "box":
		default:
			return errors.New("unsupported question type; expected noul, choice, score or box")
		}
	}
	return nil
}

type DecisionsResponse struct {
	Model   string                     `json:"model"`
	Answers map[string]json.RawMessage `json:"answers"`
	Usage   *DecisionsUsage            `json:"usage"`
}

type DecisionsUsage struct {
	InputTokens  *int64 `json:"input_tokens"`
	OutputTokens *int64 `json:"output_tokens"`
}

func validDecisionDescription(raw json.RawMessage, allowNull bool) bool {
	switch common.GetJsonType(raw) {
	case "string", "object", "array":
		return true
	case "null":
		return allowNull
	}
	return false
}
