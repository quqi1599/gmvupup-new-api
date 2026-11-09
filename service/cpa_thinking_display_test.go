package service

import (
	"fmt"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	"github.com/QuantumNous/new-api/dto"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	"github.com/tidwall/gjson"
)

func TestClaudeToCPAChatPreservesThinkingDisplay(t *testing.T) {
	for _, display := range []string{"summarized", "omitted"} {
		for _, trusted := range []bool{false, true} {
			var request dto.ClaudeRequest
			if err := common.UnmarshalJsonStr(`{"model":"deepseek-v4.1-flash","max_tokens":128,"thinking":{"type":"adaptive","display":"`+display+`"},"output_config":{"effort":"max"},"messages":[{"role":"user","content":"OK"}]}`, &request); err != nil {
				t.Fatal(err)
			}
			info := &relaycommon.RelayInfo{ChannelMeta: &relaycommon.ChannelMeta{ChannelType: constant.ChannelTypeOpenAI, ChannelOtherSettings: dto.ChannelOtherSettings{CPAAuditIdentityEnabled: trusted}}}
			out, err := ClaudeToOpenAIRequest(request, info)
			if err != nil {
				t.Fatal(err)
			}
			wire, err := common.Marshal(out)
			if err != nil {
				t.Fatal(err)
			}
			if !trusted {
				if gjson.GetBytes(wire, "thinking").Exists() {
					t.Fatal("unrelated upstream inherited CPA controls")
				}
				continue
			}
			if gjson.GetBytes(wire, "thinking.display").String() != display || gjson.GetBytes(wire, "thinking.type").String() != "adaptive" || out.ReasoningEffort != "max" {
				t.Fatalf("CPA controls lost: %s", wire)
			}
			if request.Thinking.Display != display {
				t.Fatal("source request mutated before channel retry")
			}
		}
	}
}

func TestCPAChatResponseRetainsExplicitThinkingSummary(t *testing.T) {
	for _, display := range []string{"summarized", "omitted", ""} {
		for _, trusted := range []bool{false, true} {
			for _, stop := range []string{"stop", "length", "tool_calls"} {
				request := &dto.ClaudeRequest{Thinking: &dto.Thinking{Type: "adaptive", Display: display}}
				info := &relaycommon.RelayInfo{Request: request, ChannelMeta: &relaycommon.ChannelMeta{ChannelOtherSettings: dto.ChannelOtherSettings{CPAAuditIdentityEnabled: trusted}}}
				if stop == "length" {
					info.ResponsesOutcome = &relaycommon.ResponsesOutcome{State: "incomplete", Reason: "max_output_tokens"}
				}
				message := dto.Message{Role: "assistant", Content: "answer", ReasoningContent: common.GetPointer("source summary")}
				if stop == "tool_calls" {
					message.SetToolCalls([]dto.ToolCallResponse{{ID: "call_fixture", Type: "function", Function: dto.FunctionResponse{Name: "lookup", Arguments: `{"n":9007199254740993}`}}})
				}
				response := &dto.OpenAITextResponse{Choices: []dto.OpenAITextResponseChoice{{Message: message, FinishReason: stop}}, Usage: dto.Usage{PromptTokens: 11, CompletionTokens: 7}}
				result := ResponseOpenAI2Claude(response, info)
				wire, err := common.Marshal(result)
				if err != nil {
					t.Fatal(err)
				}
				n := 0
				if trusted && display == "summarized" {
					if gjson.GetBytes(wire, "content.0.type").String() != "thinking" || gjson.GetBytes(wire, "content.0.thinking").String() != "source summary" {
						t.Fatalf("summary lost: %s", wire)
					}
					n = 1
				}
				if len(result.Content) != n+1 && stop != "tool_calls" {
					t.Fatalf("unexpected blocks: %s", wire)
				}
				if result.Content[n].Type != "text" || *result.Content[n].Text != "answer" || result.Usage.InputTokens != 11 || result.Usage.OutputTokens != 7 || result.StopReason != stopReasonOpenAI2Claude(stop) {
					t.Fatalf("answer/usage/terminal changed: %s", wire)
				}
				if stop == "tool_calls" && (len(result.Content) != n+2 || gjson.GetBytes(wire, "content."+fmt.Sprint(n+1)+".input.n").Raw != "9007199254740993") {
					t.Fatalf("tool changed: %s", wire)
				}
			}
		}
	}
}
