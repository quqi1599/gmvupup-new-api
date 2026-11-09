package service

import (
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
