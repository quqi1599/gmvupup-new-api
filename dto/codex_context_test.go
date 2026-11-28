package dto

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/tidwall/gjson"
)

func TestCodexContextPreservesToolHistoryAndPages(t *testing.T) {
	r := OpenAIResponsesRequest{Input: json.RawMessage(`[
	 {"type":"function_call_output","namespace":"agent","name":"codex_apps_open_page","output":"{\"page_id\":null}"},
	 {"type":"function_call_output","namespace":"agent","name":"codex_apps_selected_page_part_1_of_2","output":"Companion Page snapshot part 1 of 2:\n{fragment"},
	 {"type":"function_call_output","namespace":"agent","name":"browser_info","call_id":"call_real","output":"real"},
	 {"type":"function_call_output","namespace":"agent","name":"unknown","output":"留在原处"},
	 {"type":"function_call_output","namespace":"agent","name":"ide_context","call_id":null,"output":"null is explicit"},
	 {"type":"function_call_output","namespace":"agent","name":"ide_context","output":"v","output":"duplicate"},
	 {"type":"function_call_output","namespace":"agent","name":"codex_apps_open_page","output":"{\"page_id\":42}"},
	 {"type":"message","metadata":{"large":9007199254740993}},
	 {"type":"function_call_output","namespace":"agent","name":"untrusted_content","output":""}
	]`)}
	require.Equal(t, 3, r.NormalizeCodexContext())
	items := gjson.ParseBytes(r.Input).Array()
	require.Equal(t, "user", items[0].Get("role").Str)
	require.Contains(t, items[0].Get("content.0.text").Str, `{"page_id":null}`)
	require.Contains(t, items[1].Get("content.0.text").Str, "{fragment")
	for _, idx := range []int{2, 3, 4, 5, 6} {
		require.Equal(t, "function_call_output", items[idx].Get("type").Str)
	}
	require.Equal(t, "call_real", items[2].Get("call_id").Str)
	require.Equal(t, "9007199254740993", items[7].Get("metadata.large").Raw)
	require.Equal(t, 0, r.NormalizeCodexContext())
}

func TestChatCodexContextOnlyConvertsQualifiedUnpairedTools(t *testing.T) {
	known, unknown := "agent.ide_context", "ide_context"
	r := GeneralOpenAIRequest{Messages: []Message{
		{Role: "tool", Name: &known, Content: "active selection"},
		{Role: "tool", Name: &known, ToolCallId: "call_1", Content: "real output"},
		{Role: "tool", Name: &unknown, Content: "unknown origin"},
	}}
	r.NormalizeCodexContext()
	require.Equal(t, "user", r.Messages[0].Role)
	require.Equal(t, "<external_ide_context>active selection</external_ide_context>", r.Messages[0].Content)
	require.Equal(t, "call_1", r.Messages[1].ToolCallId)
	require.Equal(t, "tool", r.Messages[1].Role)
	require.Equal(t, "tool", r.Messages[2].Role)
}
