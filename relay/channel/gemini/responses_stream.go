package gemini

import (
	"net/http"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/dto"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	"github.com/QuantumNous/new-api/relay/helper"
	"github.com/QuantumNous/new-api/service/openaicompat"
	"github.com/QuantumNous/new-api/types"

	"github.com/gin-gonic/gin"
)

func geminiResponseID(c *gin.Context) string {
	id := c.GetString(common.RequestIdKey)
	if id == "" {
		id = common.GetUUID()
	}
	return "resp_" + id
}

func decorateGeminiResponse(info *relaycommon.RelayInfo, response *dto.OpenAIResponsesResponse) {
	req, ok := info.Request.(*dto.OpenAIResponsesRequest)
	if !ok || req == nil {
		return
	}
	response.Metadata = req.Metadata
	response.Instructions = req.Instructions
	response.ToolChoice = req.ToolChoice
	response.ParallelToolCalls = true
	response.Reasoning = req.Reasoning
	if req.Temperature != nil {
		response.Temperature = *req.Temperature
	}
	if req.TopP != nil {
		response.TopP = *req.TopP
	}
	if req.MaxOutputTokens != nil {
		response.MaxOutputTokens = int(*req.MaxOutputTokens)
	}
	if len(req.Tools) > 0 {
		_ = common.UnmarshalWithNumbers(req.Tools, &response.Tools)
	}
}

func geminiResponsesStreamHandler(c *gin.Context, info *relaycommon.RelayInfo, resp *http.Response) (*dto.Usage, *types.NewAPIError) {
	state := openaicompat.NewChatToResponsesStreamState(geminiResponseID(c), info.UpstreamModelName)
	var chunks geminiChunker
	var writeErr error
	var invalid *types.NewAPIError
	seq := 0
	send := func(p dto.ResponsesStreamResponse) bool {
		if p.Response != nil {
			decorateGeminiResponse(info, p.Response)
		}
		wire, err := common.Marshal(struct {
			dto.ResponsesStreamResponse
			Sequence int `json:"sequence_number"`
		}{p, seq})
		if err == nil {
			helper.ExtendWriteDeadline(c)
			err = helper.ResponseChunkData(c, p, string(wire))
		}
		if err != nil {
			writeErr = err
			return false
		}
		seq++
		info.SendResponseCount++
		if p.Response != nil {
			info.ObserveResponsesTerminal(p.Type, p.Response)
		}
		return true
	}
	events := func(list []openaicompat.ChatToResponsesStreamEvent) bool {
		for _, ev := range list {
			p := ev.Payload
			if p.Type == "response.output_text.done" && p.Text != nil {
				if !send(p) {
					return false
				}
				part := dto.ResponsesStreamResponse{Type: "response.content_part.done", ItemID: p.ItemID, OutputIndex: p.OutputIndex, ContentIndex: p.ContentIndex, Part: &dto.ResponsesReasoningSummaryPart{Type: "output_text", Text: *p.Text, Annotations: common.GetPointer([]any{})}}
				if !send(part) {
					return false
				}
				continue
			}
			if !send(p) {
				return false
			}
			if p.Type == "response.output_item.added" && p.Item != nil && p.Item.Type == "message" {
				if !send(dto.ResponsesStreamResponse{Type: "response.content_part.added", ItemID: p.Item.ID, OutputIndex: p.OutputIndex, ContentIndex: common.GetPointer(0), Part: &dto.ResponsesReasoningSummaryPart{Type: "output_text", Text: "", Annotations: common.GetPointer([]any{})}}) {
					return false
				}
			}
		}
		return true
	}
	usage, streamErr := geminiStreamHandler(c, info, resp, func(_ string, frame *dto.GeminiChatResponse) bool {
		if len(frame.Candidates) > 1 || (len(frame.Candidates) == 1 && frame.Candidates[0].Index != 0) {
			invalid = geminiBadResult("multiple Gemini candidates cannot share one Responses output")
			return false
		}
		chunk := chunks.chunk(frame)
		list, e := openaicompat.ChatCompletionsStreamChunkToResponsesEvents(chunk, state)
		if e != nil {
			invalid = geminiBadResult("could not convert Gemini stream event")
			return false
		}
		return events(list)
	})
	if writeErr != nil {
		return usage, geminiWriteError(c, info, writeErr)
	}
	if invalid != nil {
		if helper.StreamStarted(c) {
			info.PartialStreamError = invalid
		}
		return usage, invalid
	}
	if streamErr != nil {
		return usage, streamErr
	}
	state.SetUsage(usage)
	if !events(openaicompat.FinalizeChatCompletionsStreamToResponses(state)) {
		return usage, geminiWriteError(c, info, writeErr)
	}
	return usage, nil
}
