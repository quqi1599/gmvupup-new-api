package websearch

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
)

type historyReadCounter struct{ reads int }

func (s *historyReadCounter) Put(context.Context, string, Record) error { return nil }
func (s *historyReadCounter) Get(context.Context, string) (Record, error) {
	s.reads++
	return Record{}, nil
}

func TestContinueResponsesOnSocketSendsOnlyNewResults(t *testing.T) {
	p := &Plan{Protocol: "responses", Body: Object{"input": Raw("prior history"), "tool_choice": Raw("required")}}
	p.ContinueResponsesOnSocket("resp_approved", []Executed{{Call: Call{ID: "call_search"}, Data: Raw("result")}})
	items := Arr(p.Body["input"])
	if len(items) != 1 || Text(p.Body["previous_response_id"]) != "resp_approved" || p.Body["tool_choice"] != nil {
		t.Fatalf("incorrect native continuation: %s", Raw(p.Body))
	}
	item, err := Obj(items[0])
	if err != nil || Text(item["type"]) != "function_call_output" || Text(item["call_id"]) != "call_search" || Text(item["output"]) != string(Raw("result")) {
		t.Fatalf("lost tool result: %s", items[0])
	}
}

func TestManagedHistoryOnlyRecognizesProtocolReferences(t *testing.T) {
	for _, tc := range []struct {
		protocol, body string
		managed        bool
	}{
		{"responses", `{"input":"ws_na_fixture"}`, false},
		{"responses", `{"input":[{"type":"message","id":"ws_na_fixture","role":"user","content":"ws_na_other"}]}`, false},
		{"responses", `{"input":[{"type":"function_call","id":"ws_na_fixture","call_id":"ws_na_function","name":"web_search","arguments":"{}"}]}`, false},
		{"responses", `{"input":[{"type":"web_search_call","id":"ws_na_fixture"}]}`, true},
		{"responses", `{"input":[{"type":"item_reference","id":"ws_na_fixture"}]}`, true},
		{"responses", `{"input":[{"type":"web_search_call","id":"ws_provider"}]}`, false},
		{"claude", `{"messages":[{"role":"assistant","content":[{"type":"text","id":"ws_na_fixture","text":"ws_na_other"}]}]}`, false},
		{"claude", `{"messages":[{"role":"assistant","content":[{"type":"tool_use","id":"ws_na_fixture","name":"web_search","input":{}},{"type":"text","tool_use_id":"ws_na_fixture","text":"ordinary"}]}]}`, false},
		{"claude", `{"messages":[{"role":"assistant","content":[{"type":"server_tool_use","id":"ws_na_fixture","name":"web_search","input":{}}]}]}`, true},
		{"claude", `{"messages":[{"role":"assistant","content":[{"type":"server_tool_use","id":"ws_na_fixture","name":"web_fetch","input":{}}]}]}`, true},
		{"claude", `{"messages":[{"role":"assistant","content":[{"type":"web_search_tool_result","tool_use_id":"ws_na_fixture","content":[]}]}]}`, true},
		{"chat", `{"messages":[{"role":"assistant","content":"ws_na_fixture"}]}`, false},
	} {
		t.Run(tc.protocol+tc.body, func(t *testing.T) {
			body, err := Obj([]byte(tc.body))
			require.NoError(t, err)
			p := &Plan{Protocol: tc.protocol, Body: body}
			require.Equal(t, tc.managed, p.HasManagedHistory())
			if !tc.managed {
				store := &historyReadCounter{}
				require.NoError(t, p.Restore(t.Context(), store))
				require.Zero(t, store.reads)
				require.JSONEq(t, tc.body, string(Raw(p.Body)))
			}
		})
	}
}

func TestManagedClaudeOrphanResultIsNotSilentlyDeleted(t *testing.T) {
	body, err := Obj([]byte(`{"messages":[{"role":"assistant","content":[{"type":"web_search_tool_result","tool_use_id":"ws_na_fixture","content":"untrusted result"}]}]}`))
	require.NoError(t, err)
	p := &Plan{Protocol: "claude", Body: body}
	require.ErrorContains(t, p.Restore(t.Context(), memoryStore{}), "missing its original call")
}
