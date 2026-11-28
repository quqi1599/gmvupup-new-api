package dto

import (
	"bytes"
	"encoding/json"
	"regexp"
	"strconv"
	"strings"

	"github.com/QuantumNous/new-api/common"
	"github.com/tidwall/gjson"
)

var codexPagePart = regexp.MustCompile("^codex_apps_selected_page_part_([1-9][0-9]*)_of_([1-9][0-9]*)$")

// Codex emits these external context fragments without a corresponding tool
// call. Match the namespace and shape before representing them as user data.
func codexContextText(name, output string) (string, bool) {
	page := false
	switch name {
	case "browser_info", "goal_objective", "untrusted_context", "ide_context":
		if strings.TrimSpace(output) == "" {
			return "", false
		}
	case "untrusted_content": // The fallback source can intentionally contain an empty string.
	case "proposed_plan":
		if output == "" {
			return "", false
		}
	case "codex_apps_open_page":
		if !codexPageSnapshot(output) {
			return "", false
		}
		page = true
	default:
		p := codexPagePart.FindStringSubmatch(name)
		if p == nil {
			return "", false
		}
		part, e1 := strconv.Atoi(p[1])
		total, e2 := strconv.Atoi(p[2])
		header, snapshot, ok := strings.Cut(output, "\n")
		if e1 != nil || e2 != nil || part > total || !ok || strings.TrimSuffix(header, "\r") != "Companion Page snapshot part "+p[1]+" of "+p[2]+":" || strings.TrimSpace(snapshot) == "" || (total == 1 && !codexPageSnapshot(snapshot)) {
			return "", false
		}
		page = true
	}
	if page {
		output = "\n" + output + "\n"
	}
	return "<external_" + name + ">" + output + "</external_" + name + ">", true
}

func uniqueCodexFields(obj gjson.Result) bool {
	seen := map[string]bool{}
	valid := true
	obj.ForEach(func(k, v gjson.Result) bool {
		if seen[k.String()] {
			valid = false
			return false
		}
		seen[k.String()] = true
		return true
	})
	return valid
}

func codexPageSnapshot(text string) bool {
	raw := gjson.Parse(text)
	if !raw.IsObject() || !gjson.Valid(text) || !uniqueCodexFields(raw) {
		return false
	}
	id := raw.Get("page_id")
	return id.Exists() && (id.Type == gjson.Null || (id.Type == gjson.String && strings.TrimSpace(id.Str) != ""))
}

// NormalizeCodexContext returns the number of external context fragments converted.
func (r *OpenAIResponsesRequest) NormalizeCodexContext() int {
	if !bytes.Contains(r.Input, []byte("function_call_output")) && !bytes.Contains(r.Input, []byte("\\u")) {
		return 0
	}
	var items []json.RawMessage
	if common.Unmarshal(r.Input, &items) != nil {
		return 0
	}
	changed := 0
	for i, item := range items {
		obj := gjson.ParseBytes(item)
		if obj.Get("type").Str != "function_call_output" || obj.Get("namespace").Str != "agent" {
			continue
		}
		id := obj.Get("call_id")
		if id.Exists() && (id.Type != gjson.String || strings.TrimSpace(id.Str) != "") {
			continue
		}
		out := obj.Get("output")
		if out.Type != gjson.String || !uniqueCodexFields(obj) {
			continue
		}
		s, ok := codexContextText(obj.Get("name").Str, out.Str)
		if !ok {
			continue
		}
		msg := map[string]any{"type": "message", "role": "user", "content": []map[string]string{{"type": "input_text", "text": s}}}
		encoded, err := common.Marshal(msg)
		if err != nil {
			return 0
		}
		items[i] = encoded
		changed++
	}
	if changed > 0 {
		input, err := common.Marshal(items)
		if err != nil {
			return 0
		}
		r.Input = input
	}
	return changed
}

// Some Chat-compatible clients use a fully qualified tool name instead of a
// separate namespace. Never reinterpret an ordinary tool or a paired result.
func (r *GeneralOpenAIRequest) NormalizeCodexContext() {
	for i := range r.Messages {
		m := &r.Messages[i]
		if m.Role != "tool" || m.ToolCallId != "" || m.Name == nil {
			continue
		}
		name, ok := strings.CutPrefix(*m.Name, "agent.")
		output, text := m.Content.(string)
		if !ok || !text {
			continue
		}
		if value, ok := codexContextText(name, output); ok {
			r.Messages[i] = Message{Role: "user", Content: value}
		}
	}
}
