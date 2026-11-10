package openai

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/dto"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	relayconstant "github.com/QuantumNous/new-api/relay/constant"
	"github.com/QuantumNous/new-api/relay/helper"
	"github.com/QuantumNous/new-api/service"
	"github.com/QuantumNous/new-api/types"
	"github.com/gin-gonic/gin"
	"github.com/tidwall/gjson"
)

type miniMaxH3Usage struct {
	dto.Usage
	GeneratedImages *int `json:"generated_images"`
}

func validateMiniMaxH3Usage(usage *miniMaxH3Usage, count int) error {
	if usage == nil || usage.GeneratedImages == nil || *usage.GeneratedImages != count || count < 0 || count > 10 {
		return fmt.Errorf("MiniMax H3 usage does not match completed images")
	}
	if usage.InputTokens < 0 || usage.OutputTokens < 0 || usage.TotalTokens < 0 {
		return fmt.Errorf("MiniMax H3 returned negative token usage")
	}
	return nil
}

// MiniMaxH3ImageHandler preserves the native H3 result and event schema. H3's
// partial_succeeded is a final image, whereas OpenAI's partial_image is only a
// preview; its completed event contains aggregate usage, not another image.
func MiniMaxH3ImageHandler(c *gin.Context, info *relaycommon.RelayInfo, resp *http.Response) (*dto.Usage, *types.NewAPIError) {
	if resp == nil || resp.Body == nil {
		return nil, imageResponseError(fmt.Errorf("invalid MiniMax H3 image response"))
	}
	if strings.Contains(strings.ToLower(resp.Header.Get("Content-Type")), "text/event-stream") {
		return miniMaxH3ImageStreamHandler(c, info, resp)
	}
	defer service.CloseResponseBodyGracefully(resp)
	body, err := readOpenAIImageJSON(resp.Body)
	if err != nil {
		return nil, imageResponseError(err)
	}
	var result struct {
		Usage *miniMaxH3Usage `json:"usage"`
	}
	if err := common.Unmarshal(body, &result); err != nil {
		return nil, imageResponseError(err)
	}
	count := 0
	for _, item := range gjson.GetBytes(body, "data").Array() {
		if item.Get("url").String() != "" || item.Get("b64_json").String() != "" {
			count++
		}
	}
	if err := validateMiniMaxH3Usage(result.Usage, count); err != nil {
		return nil, imageResponseError(err)
	}
	// Reuse canonical token normalization, URL caching and delivery accounting.
	copyResp := *resp
	copyResp.Body = io.NopCloser(bytes.NewReader(body))
	return OpenaiImageHandler(c, info, &copyResp)
}

func miniMaxH3ImageStreamHandler(c *gin.Context, info *relaycommon.RelayInfo, resp *http.Response) (usage *dto.Usage, apiErr *types.NewAPIError) {
	usage = &dto.Usage{}
	c.Set(imageCompletedCountKey, 0)
	c.Set(imageReportedUsageKey, false)
	streamOutput := info.IsStream
	if request, ok := info.Request.(*dto.ImageRequest); ok {
		streamOutput = request.IsStream(c)
	}
	previousDisablePing := info.DisablePing
	if !streamOutput {
		info.DisablePing = true
	}
	defer func() {
		info.DisablePing = previousDisablePing
		if apiErr != nil {
			types.ErrOptionWithSkipRetry()(apiErr)
			// Even if downstream delivery failed or buffering produced no output,
			// confirmed images are billable; never refund them then replay.
			info.PartialStreamError = apiErr
		}
	}()
	prefix := "image_generation."
	if info.RelayMode == relayconstant.RelayModeImagesEdits {
		prefix = "image_edit."
	}
	images := map[int]json.RawMessage{}
	succeeded := map[int]bool{}
	var summary []byte
	var outputFormat string
	var failure *types.NewAPIError
	helper.StreamScannerHandlerWithDecoder(c, resp, info, &imageSSEDecoder{}, func(frame helper.StreamFrame, sr *helper.StreamResult) {
		if frame.Kind == "eof" {
			return // EOF without the terminal event remains a failed stream.
		}
		raw := []byte(frame.Data)
		var event struct {
			Type         string          `json:"type"`
			ImageIndex   *int            `json:"image_index"`
			URL          string          `json:"url"`
			Code         string          `json:"code"`
			Message      string          `json:"message"`
			OutputFormat string          `json:"output_format"`
			Usage        *miniMaxH3Usage `json:"usage"`
			Error        json.RawMessage `json:"error"`
		}
		if err := common.Unmarshal(raw, &event); err != nil {
			sr.Stop(fmt.Errorf("invalid MiniMax H3 image event: %w", err))
			return
		}
		name := event.Type
		if name == "" {
			name = frame.Event
		}
		terminal := name == prefix+"completed" || name == "error"
		switch name {
		case prefix + "partial_succeeded", prefix + "partial_failed":
			if event.ImageIndex == nil || *event.ImageIndex < 0 || *event.ImageIndex >= 10 ||
				(name == prefix+"partial_succeeded" && event.URL == "") ||
				(name == prefix+"partial_failed" && (event.Code == "" || event.Message == "")) {
				sr.Stop(fmt.Errorf("invalid MiniMax H3 per-image result"))
				return
			}
			index := *event.ImageIndex
			if _, present := images[index]; present {
				if succeeded[index] != (name == prefix+"partial_succeeded") {
					sr.Stop(fmt.Errorf("conflicting MiniMax H3 per-image result"))
				}
				return // Retransmitted events must not multiply images or charges.
			}
			var item map[string]json.RawMessage
			if name == prefix+"partial_succeeded" {
				item = make(map[string]json.RawMessage)
				for _, key := range []string{"url", "size", "revised_prompt"} {
					if value := gjson.GetBytes(raw, key); value.Exists() {
						item[key] = json.RawMessage(value.Raw)
					}
				}
				succeeded[index] = true
				outputFormat = event.OutputFormat
			} else {
				detail, _ := common.Marshal(map[string]string{"code": event.Code, "message": event.Message})
				item = map[string]json.RawMessage{"error": detail}
			}
			images[index], _ = common.Marshal(item)
			c.Set(imageCompletedCountKey, len(succeeded))
		case prefix + "completed", "error":
			if err := validateMiniMaxH3Usage(event.Usage, len(succeeded)); err != nil {
				sr.Stop(err)
				return
			}
			if (name == prefix+"completed" && len(succeeded) == 0) ||
				(name == "error" && (len(succeeded) != 0 || len(event.Error) == 0 || string(event.Error) == "null")) {
				sr.Stop(fmt.Errorf("invalid MiniMax H3 terminal event"))
				return
			}
			usage = &event.Usage.Usage
			normalizeOpenAIImageUsage(usage, gjson.GetBytes(raw, "usage").Raw)
			c.Set(imageReportedUsageKey, true)
			summary = raw
			if name == "error" {
				failure = imageResponseError(fmt.Errorf("MiniMax H3 image generation failed"))
			}
		default:
			sr.Stop(fmt.Errorf("unknown MiniMax H3 image event"))
			return
		}
		if !sr.Accept() {
			return
		}
		if streamOutput {
			if name == prefix+"partial_succeeded" {
				raw = service.RewriteImageResultURLs(c, raw)
			}
			if err := writeOpenAIImageEvent(c, name, string(raw)); err != nil {
				failure = helper.DownstreamStreamError(c, info, err)
				sr.Stop(err)
				return
			}
		}
		if terminal {
			sr.Done()
		}
	})
	applyUsagePostProcessing(info, usage, summary)
	if failure != nil {
		return usage, failure
	}
	if err := helper.PreOutputStreamError(c, info); err != nil {
		return usage, err
	}
	if err := helper.PostOutputStreamError(c, info); err != nil {
		return usage, err
	}
	if len(summary) == 0 {
		return usage, imageResponseError(fmt.Errorf("MiniMax H3 image stream ended without usage summary"))
	}
	if !streamOutput {
		data := make([]json.RawMessage, 0, len(images))
		for index := 0; index < 10; index++ {
			if item, exists := images[index]; exists {
				data = append(data, item)
			}
		}
		body, err := common.Marshal(map[string]any{
			"created": gjson.GetBytes(summary, "created").Int(), "data": data,
			"output_format": outputFormat, "request_id": gjson.GetBytes(summary, "request_id").String(),
			"usage": json.RawMessage(gjson.GetBytes(summary, "usage").Raw),
		})
		if err != nil {
			return usage, imageResponseError(err)
		}
		resp.Header.Set("Content-Type", "application/json")
		return usage, writeOpenAIImageJSON(c, info, resp, body)
	}
	return usage, nil
}
