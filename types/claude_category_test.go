package types

import (
	"github.com/stretchr/testify/require"
	"testing"
)

func TestClaudeErrorUsesStructuredCategory(t *testing.T) {
	for _, test := range []struct {
		kind string
		code any
		want string
	}{
		{"rate_limit_error", nil, "rate_limit_error"},
		{"invalid_request_error", "invalid_value", "invalid_request_error"},
		{"", "overloaded_error", "overloaded_error"},
		{"", 429, "upstream_error"},
		{"", "internal_custom_code", "upstream_error"},
	} {
		got := WithOpenAIError(OpenAIError{Type: test.kind, Code: test.code, Message: "synthetic"}, 429).ToClaudeError()
		require.Equal(t, test.want, got.Type)
		require.Equal(t, test.code, got.Code)
	}
	got := WithOpenAIError(OpenAIError{Type: "invalid_request_error", Code: "request_too_large"}, 413).ToClaudeError()
	require.Equal(t, "invalid_request_error", got.Type)
	require.Equal(t, "request_too_large", got.Code)
}
