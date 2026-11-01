package service

import (
	"strings"

	"github.com/QuantumNous/new-api/types"
	"github.com/tidwall/gjson"
)

// Match the complete structured error message, never a substring in raw JSON,
// debug output, ordinary parameter errors or a content-policy rejection.
func markUpstreamClientRejection(err *types.NewAPIError, message string, body []byte) {
	if err == nil || types.IsSkipRetryError(err) || !strings.EqualFold(strings.TrimSpace(message), "Illegal API invocation from an unapproved channel") {
		return
	}
	switch err.GetErrorCode() {
	case types.ErrorCodeBadResponseStatusCode, types.ErrorCodeInvalidRequest, "unknown_error", "invalid_request_error", "11128", "11140":
	default:
		return
	}
	switch err.ToOpenAIError().Type {
	case "", "bad_response_status_code", "invalid_request", "invalid_request_error", "upstream_error", "api_error":
	default:
		return
	}
	// The generic error DTO does not retain flat-envelope codes/types. Check
	// them before granting a retry so a policy/state rejection cannot lose its
	// terminal meaning merely because the provider uses a different envelope.
	root := gjson.ParseBytes(body)
	for _, path := range []string{"code", "error.code"} {
		switch root.Get(path).String() {
		case "", "bad_response_status_code", "invalid_request", "unknown_error", "invalid_request_error", "11128", "11140":
		default:
			return
		}
	}
	for _, path := range []string{"type", "error.type"} {
		switch root.Get(path).String() {
		case "", "error", "bad_response_status_code", "invalid_request", "invalid_request_error", "upstream_error", "api_error":
		default:
			return
		}
	}
	matched := false
	for _, path := range []string{"error.message", "message", "msg", "error"} {
		value := root.Get(path)
		matched = matched || value.Type == gjson.String && strings.TrimSpace(value.String()) == strings.TrimSpace(message)
	}
	if !matched {
		return
	}
	err.MarkUpstreamClientRejected()
}
