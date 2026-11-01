package service

import (
	"context"
	"errors"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/setting/operation_setting"
	"github.com/QuantumNous/new-api/types"
	"github.com/stretchr/testify/require"
)

const clientRejectionMessage = "Illegal API invocation from an unapproved channel"

func TestUpstreamClientRejectionRequiresExactHTTPContract(t *testing.T) {
	for _, body := range []string{
		`{"message":"` + clientRejectionMessage + `"}`,
		`{"code":11128,"msg":"` + clientRejectionMessage + `"}`,
		`{"error":{"type":"invalid_request_error","code":null,"message":"` + clientRejectionMessage + `"}}`,
		`{"error":{"type":"invalid_request_error","code":11128,"message":"` + clientRejectionMessage + `"}}`,
		`{"error":{"type":"api_error","message":"  illegal api invocation from an unapproved channel  "}}`,
	} {
		for _, showBody := range []bool{false, true} {
			err := RelayErrorHandler(context.Background(), &http.Response{StatusCode: 400, Body: io.NopCloser(strings.NewReader(body))}, showBody)
			require.True(t, types.IsUpstreamClientRejected(err), body)
			require.Equal(t, types.ErrorCodeUpstreamClientRejected, err.GetErrorCode())
			require.Equal(t, 400, err.StatusCode)
			require.False(t, types.IsSkipRetryError(err))
			require.False(t, types.IsChannelError(err), "must not bypass normal retry/specific-channel checks")
			require.False(t, types.IsChannelPenaltyAllowed(err))
			require.Equal(t, string(types.ErrorCodeUpstreamClientRejected), err.ToOpenAIError().Code)
		}
	}
	for name, body := range map[string]string{
		"quoted_message":    `{"message":"client quoted: ` + clientRejectionMessage + `"}`,
		"other_field":       `{"message":"invalid tools","detail":"` + clientRejectionMessage + `"}`,
		"detail_only":       `{"detail":"` + clientRejectionMessage + `"}`,
		"flat_policy_code":  `{"code":"content_policy_violation","message":"` + clientRejectionMessage + `"}`,
		"flat_policy_type":  `{"type":"content_policy_violation","message":"` + clientRejectionMessage + `"}`,
		"flat_state_code":   `{"code":"state_owner_unavailable","msg":"` + clientRejectionMessage + `"}`,
		"flat_unknown_code": `{"code":12345,"message":"` + clientRejectionMessage + `"}`,
		"code_only":         `{"error":{"code":"upstream_client_rejected","message":"ordinary error"}}`,
		"numeric_only":      `{"code":11128,"msg":"different reason"}`,
		"audit_code":        `{"error":{"code":"cpa_content_audit_blocked","message":"` + clientRejectionMessage + `"}}`,
		"content_code":      `{"error":{"code":"content_policy_violation","message":"` + clientRejectionMessage + `"}}`,
		"content_type":      `{"error":{"type":"content_policy_violation","message":"` + clientRejectionMessage + `"}}`,
		"state_code":        `{"error":{"code":"state_owner_unavailable","message":"` + clientRejectionMessage + `"}}`,
		"invalid_json":      clientRejectionMessage,
	} {
		t.Run(name, func(t *testing.T) {
			err := RelayErrorHandler(context.Background(), &http.Response{StatusCode: 400, Body: io.NopCloser(strings.NewReader(body))}, true)
			require.False(t, types.IsUpstreamClientRejected(err))
		})
	}
	for _, status := range []int{200, 401, 403, 422, 429, 500, 502} {
		err := RelayErrorHandler(context.Background(), &http.Response{StatusCode: status, Body: io.NopCloser(strings.NewReader(`{"message":"` + clientRejectionMessage + `"}`))}, false)
		require.False(t, types.IsUpstreamClientRejected(err), "status=%d", status)
	}
	err := RelayErrorHandler(context.Background(), &http.Response{StatusCode: 400, Header: http.Header{"X-Cpa-Local-Guard": []string{"content-audit"}}, Body: io.NopCloser(strings.NewReader(`{"message":"` + clientRejectionMessage + `"}`))}, false)
	require.True(t, types.IsSkipRetryError(err))
	require.False(t, types.IsUpstreamClientRejected(err))
	err = RelayErrorHandler(context.Background(), &http.Response{StatusCode: 400, Body: &relayErrorReadCloser{err: errors.New("fixture truncated body")}}, false)
	require.False(t, types.IsUpstreamClientRejected(err))
}

func TestUpstreamClientRejectionDoesNotDisableHealthyChannel(t *testing.T) {
	oldEnabled, oldKeywords, oldRanges := common.AutomaticDisableChannelEnabled, operation_setting.AutomaticDisableKeywords, operation_setting.AutomaticDisableStatusCodeRanges
	t.Cleanup(func() {
		common.AutomaticDisableChannelEnabled, operation_setting.AutomaticDisableKeywords, operation_setting.AutomaticDisableStatusCodeRanges = oldEnabled, oldKeywords, oldRanges
	})
	common.AutomaticDisableChannelEnabled = true
	operation_setting.AutomaticDisableKeywords = []string{"illegal api invocation"}
	operation_setting.AutomaticDisableStatusCodeRanges = []operation_setting.StatusCodeRange{{Start: 400, End: 599}}
	err := RelayErrorHandler(context.Background(), &http.Response{StatusCode: 400, Body: io.NopCloser(strings.NewReader(`{"message":"` + clientRejectionMessage + `"}`))}, false)
	ResetStatusCode(err, `{"400":503}`)
	require.True(t, types.IsUpstreamClientRejected(err), "classification survives configured status mapping")
	require.False(t, ShouldDisableChannel(err))
	require.False(t, ShouldCountChannelCircuitFailure(err))
	ordinary := types.NewErrorWithStatusCode(errors.New("upstream failure"), types.ErrorCodeBadResponseStatusCode, 503)
	require.True(t, ShouldDisableChannel(ordinary))
	require.True(t, ShouldCountChannelCircuitFailure(ordinary))
}
