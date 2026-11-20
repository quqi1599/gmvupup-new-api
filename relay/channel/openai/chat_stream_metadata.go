package openai

// chatStreamMetadataOnly is deliberately conservative: unknown nonempty delta
// fields retain the existing usage behavior. Audio, refusal and legacy function
// output must not be mistaken for an empty stream just because the text-token
// counter does not understand them.
func chatStreamMetadataOnly(event map[string]any) bool {
	choices, exists := event["choices"]
	if !exists || choices == nil {
		return true
	}
	items, ok := choices.([]any)
	if !ok {
		return false
	}
	for _, item := range items {
		choice, ok := item.(map[string]any)
		if !ok {
			return false
		}
		if !emptyChatStreamValue(choice["text"]) {
			return false
		}
		if delta, ok := choice["delta"].(map[string]any); ok {
			for key, value := range delta {
				if key != "role" && !emptyChatStreamValue(value) {
					return false
				}
			}
		} else if !emptyChatStreamValue(choice["delta"]) {
			return false
		}
	}
	return true
}

func emptyChatStreamValue(value any) bool {
	switch value := value.(type) {
	case nil:
		return true
	case string:
		return value == ""
	case []any:
		return len(value) == 0
	case map[string]any:
		return len(value) == 0
	default:
		return false
	}
}

// Only recognized, nonterminal Chat deltas can bypass last-frame processing.
// Accounting treats unknown output conservatively, which is a different test.
func hasGeneratedChatDelta(event map[string]any) bool {
	choices, _ := event["choices"].([]any)
	hasOutput := false
	for _, entry := range choices {
		choice, ok := entry.(map[string]any)
		if !ok || !emptyChatStreamValue(choice["finish_reason"]) {
			return false
		}
		delta, ok := choice["delta"].(map[string]any)
		if !ok {
			return false
		}
		for _, field := range []string{"content", "reasoning_content", "reasoning"} {
			if text, _ := delta[field].(string); text != "" {
				hasOutput = true
			}
		}
		if calls, _ := delta["tool_calls"].([]any); len(calls) > 0 {
			hasOutput = true
		}
	}
	return hasOutput
}
