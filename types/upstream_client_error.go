package types

import "net/http"

// MarkUpstreamClientRejected is called only after the HTTP error parser has
// recognized the complete provider refusal. A wire code alone is not proof.
func (e *NewAPIError) MarkUpstreamClientRejected() {
	if e == nil || !e.HasUpstreamResponse() || e.StatusCode != http.StatusBadRequest || e.skipRetry {
		return
	}
	e.upstreamClientRejected = true
	e.errorCode = ErrorCodeUpstreamClientRejected
	switch detail := e.RelayError.(type) {
	case OpenAIError:
		detail.Code, detail.Type = string(e.errorCode), "upstream_error"
		e.RelayError = detail
	case ClaudeError:
		detail.Code, detail.Type = string(e.errorCode), "api_error"
		e.RelayError = detail
	}
}

func IsUpstreamClientRejected(e *NewAPIError) bool {
	return e != nil && e.upstreamClientRejected && e.HasUpstreamResponse()
}
