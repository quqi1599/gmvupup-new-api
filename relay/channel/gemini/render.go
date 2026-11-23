package gemini

import (
	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	"github.com/QuantumNous/new-api/dto"
	"strings"
)

func renderGeminiParts(parts []dto.GeminiPart) (string, string, []dto.ToolCallResponse) {
	var visible, thought strings.Builder
	lastPlain := true
	appendVisible := func(s string, plain bool) {
		if s == "" {
			return
		}
		if visible.Len() > 0 && !(plain && lastPlain) {
			visible.WriteByte('\n')
		}
		visible.WriteString(s)
		lastPlain = plain
	}
	var calls []dto.ToolCallResponse
	for i := range parts {
		p := &parts[i]
		switch {
		case p.FunctionCall != nil:
			if call := getResponseToolCall(p); call != nil {
				calls = append(calls, *call)
			}
		case p.Thought:
			thought.WriteString(p.Text)
		case p.InlineData != nil:
			label := "[media]"
			if strings.HasPrefix(p.InlineData.MimeType, "image/") {
				label = "![image]"
			}
			appendVisible(label+"(data:"+p.InlineData.MimeType+";base64,"+p.InlineData.Data+")", false)
		case p.ExecutableCode != nil:
			appendVisible("```"+p.ExecutableCode.Language+"\n"+p.ExecutableCode.Code+"\n```", false)
		case p.CodeExecutionResult != nil:
			appendVisible("```output\n"+p.CodeExecutionResult.Output+"\n```", false)
		default:
			appendVisible(p.Text, true)
		}
	}
	return visible.String(), thought.String(), calls
}

func geminiFinish(reason *string, tools bool) string {
	if reason == nil {
		return ""
	}
	switch *reason {
	case "", "FINISH_REASON_UNSPECIFIED":
		return ""
	case "STOP":
		if tools {
			return constant.FinishReasonToolCalls
		}
		return constant.FinishReasonStop
	case "MAX_TOKENS", "CONTINUATION":
		return constant.FinishReasonLength
	default:
		return constant.FinishReasonContentFilter
	}
}

// A native Gemini function call is one complete part, not an OpenAI argument
// delta. Assign each part a distinct index; remember calls across STOP frames.
type geminiChunker struct{ nextTool map[int]int }

func (s *geminiChunker) chunk(frame *dto.GeminiChatResponse) *dto.ChatCompletionsStreamResponse {
	if s.nextTool == nil {
		s.nextTool = map[int]int{}
	}
	out := &dto.ChatCompletionsStreamResponse{Object: "chat.completion.chunk"}
	for _, candidate := range frame.Candidates {
		idx := int(candidate.Index)
		text, thought, calls := renderGeminiParts(candidate.Content.Parts)
		d := dto.ChatCompletionsStreamResponseChoiceDelta{}
		if text != "" {
			d.SetContentString(text)
		}
		if thought != "" {
			d.SetReasoningContent(thought)
		}
		for k := range calls {
			calls[k].SetIndex(s.nextTool[idx])
			s.nextTool[idx]++
		}
		d.ToolCalls = calls
		choice := dto.ChatCompletionsStreamResponseChoice{Index: idx, Delta: d}
		if reason := geminiFinish(candidate.FinishReason, s.nextTool[idx] > 0); reason != "" {
			choice.FinishReason = common.GetPointer(reason)
		}
		out.Choices = append(out.Choices, choice)
	}
	return out
}
