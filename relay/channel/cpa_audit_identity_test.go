package channel

import (
	"net/http"
	"testing"
	"time"

	common2 "github.com/QuantumNous/new-api/common"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	"github.com/QuantumNous/new-api/types"
	"github.com/stretchr/testify/require"
)

func TestApplyCPAAuditIdentityHeadersSignsVerifiedCustomerMetadata(t *testing.T) {
	header := http.Header{
		"X-CPA-Audit-Signature": []string{"spoofed"},
		"X-CPA-Audit-User-Id":   []string{"999"},
	}
	info := &relaycommon.RelayInfo{
		UserId:          42,
		TokenId:         73,
		TokenName:       "production-token",
		RequestId:       "req-abc",
		OriginModelName: "gpt-5.4",
		ChannelMeta: &relaycommon.ChannelMeta{
			UpstreamModelName: "gpt-5.4-cpa",
		},
	}
	info.ChannelOtherSettings.CPAAuditIdentityEnabled = true
	now := time.Unix(1_800_000_000, 0)

	err := applyCPAAuditIdentityHeaders(header, http.MethodPost, "https://cpa.example/v1/responses?beta=true", info, now, "shared-secret")
	require.NoError(t, err)
	require.Equal(t, "1", header.Get(cpaAuditHeaderVersion))
	require.Equal(t, "42", header.Get(cpaAuditHeaderUserID))
	require.Equal(t, "73", header.Get(cpaAuditHeaderTokenID))
	require.Equal(t, "production-token", header.Get(cpaAuditHeaderTokenName))
	require.Equal(t, "req-abc", header.Get(cpaAuditHeaderRequestID))
	require.Equal(t, "1800000000", header.Get(cpaAuditHeaderTimestamp))
	require.Equal(t, "gpt-5.4-cpa", header.Get(cpaAuditHeaderModel))

	canonical := cpaAuditCanonicalValues(
		"1",
		"1800000000",
		"req-abc",
		"42",
		"73",
		"production-token",
		http.MethodPost,
		"/v1/responses",
		"gpt-5.4-cpa",
		"0",
	)
	require.Equal(t, common2.HmacSha256(canonical, "shared-secret"), header.Get(cpaAuditHeaderSignature))
	require.NotEqual(t, "spoofed", header.Get(cpaAuditHeaderSignature))
}

func TestApplyCPAAuditIdentityHeadersStripsSpoofedHeadersWhenDisabled(t *testing.T) {
	header := http.Header{
		"X-CPA-Audit-Signature": []string{"spoofed"},
		"X-CPA-Audit-User-Id":   []string{"999"},
		"X-Unrelated":           []string{"kept"},
	}

	err := applyCPAAuditIdentityHeaders(header, http.MethodPost, "https://provider.example/v1/responses", &relaycommon.RelayInfo{ChannelMeta: &relaycommon.ChannelMeta{}}, time.Now(), "")
	require.NoError(t, err)
	require.Empty(t, header.Get(cpaAuditHeaderSignature))
	require.Empty(t, header.Get(cpaAuditHeaderUserID))
	require.Equal(t, "kept", header.Get("X-Unrelated"))
}

func TestApplyCPAAuditIdentityHeadersFailsClosedWithoutSecret(t *testing.T) {
	info := &relaycommon.RelayInfo{
		UserId:      42,
		TokenId:     73,
		RequestId:   "req-abc",
		ChannelMeta: &relaycommon.ChannelMeta{},
	}
	info.ChannelOtherSettings.CPAAuditIdentityEnabled = true

	err := applyCPAAuditIdentityHeaders(http.Header{}, http.MethodPost, "https://cpa.example/v1/responses", info, time.Now(), "")
	require.Error(t, err)
	var apiErr *types.NewAPIError
	require.ErrorAs(t, err, &apiErr)
	require.True(t, types.IsSkipRetryError(apiErr))
	require.Equal(t, types.ErrorCodeChannelCPAAuditIdentityInvalid, apiErr.GetErrorCode())
}

func TestApplyCPAAuditIdentityHeadersSignsChannelTestsWithoutCustomerIdentity(t *testing.T) {
	header := http.Header{"X-CPA-Audit-Signature": []string{"spoofed"}}
	info := &relaycommon.RelayInfo{
		IsChannelTest: true,
		ChannelMeta:   &relaycommon.ChannelMeta{},
	}
	info.ChannelOtherSettings.CPAAuditIdentityEnabled = true

	err := applyCPAAuditIdentityHeaders(header, http.MethodPost, "https://cpa.example/v1/chat/completions", info, time.Now(), "shared-secret")
	require.NoError(t, err)
	require.Equal(t, "0", header.Get(cpaAuditHeaderUserID))
	require.Equal(t, "0", header.Get(cpaAuditHeaderTokenID))
	require.Equal(t, "__channel_test__", header.Get(cpaAuditHeaderTokenName))
	require.Equal(t, "1", header.Get(cpaAuditHeaderChannelTest))
	require.NotEmpty(t, header.Get(cpaAuditHeaderSignature))
}

func TestShouldSkipPassthroughHeaderRejectsCPAIdentityNamespace(t *testing.T) {
	require.True(t, shouldSkipPassthroughHeader("X-CPA-Audit-User-ID"))
	require.True(t, shouldSkipPassthroughHeader("x-cpa-audit-signature"))
}

func TestPlaygroundAuditIdentityV2GoldenContract(t *testing.T) {
	info := &relaycommon.RelayInfo{UserId: 42, IsPlayground: true, RequestId: "req-pg", OriginModelName: "claude-sonnet-4-6", ChannelMeta: &relaycommon.ChannelMeta{}}
	info.ChannelOtherSettings.CPAAuditIdentityEnabled = true
	header := http.Header{"x-cpa-audit-user-id": {"999"}, "X-CPA-Audit-Identity-Kind": {"forged"}, "X-CPA-Audit-Channel-Test": {"1"}}
	require.NoError(t, applyCPAAuditIdentityHeaders(header, "POST", "https://cpa.example/v1/chat/completions", info, time.Unix(1800000000, 0), "shared-secret"))
	require.Equal(t, "2", header.Get(cpaAuditHeaderVersion))
	require.Equal(t, "playground", header.Get(cpaAuditHeaderIdentityKind))
	require.Equal(t, "42", header.Get(cpaAuditHeaderUserID))
	require.Equal(t, "0", header.Get(cpaAuditHeaderTokenID))
	require.Equal(t, "__playground__", header.Get(cpaAuditHeaderTokenName))
	require.Empty(t, header.Get(cpaAuditHeaderChannelTest))
	require.NotContains(t, header, "x-cpa-audit-user-id")
	// Independently computed fixed vector, also consumed by CPA's verifier test.
	require.Equal(t, "d488ce87fc05402b1e47dee1eca3bc0850d859335bf912fe8f324ec393f74e78", header.Get(cpaAuditHeaderSignature))
}

func TestPlaygroundAuditIdentityRejectsMissingOrConflictingIdentity(t *testing.T) {
	for _, tc := range []struct {
		name   string
		change func(*relaycommon.RelayInfo)
	}{
		{"no session user", func(i *relaycommon.RelayInfo) { i.UserId = 0 }},
		{"negative token", func(i *relaycommon.RelayInfo) { i.TokenId = -1 }},
		{"real token", func(i *relaycommon.RelayInfo) { i.TokenId = 73 }},
		{"channel test", func(i *relaycommon.RelayInfo) { i.IsChannelTest = true }},
		{"missing request", func(i *relaycommon.RelayInfo) { i.RequestId = "" }},
		{"empty after sanitizing", func(i *relaycommon.RelayInfo) { i.RequestId = "\r\n\x00" }},
		{"ordinary request without token", func(i *relaycommon.RelayInfo) { i.IsPlayground = false }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			info := &relaycommon.RelayInfo{UserId: 42, IsPlayground: true, RequestId: "req-pg", ChannelMeta: &relaycommon.ChannelMeta{}}
			info.ChannelOtherSettings.CPAAuditIdentityEnabled = true
			tc.change(info)
			header := http.Header{"X-CPA-Audit-Signature": {"forged"}, "x-cpa-audit-identity-kind": {"playground"}}
			err := applyCPAAuditIdentityHeaders(header, "POST", "https://cpa.example/v1/chat/completions", info, time.Now(), "shared-secret")
			require.Error(t, err)
			require.Empty(t, header)
		})
	}
}

func TestManagedWebAuditPreservesPlaygroundIdentity(t *testing.T) {
	old := common2.CPAAuditIdentitySecret
	common2.CPAAuditIdentitySecret = "synthetic-managed-web-secret"
	t.Cleanup(func() { common2.CPAAuditIdentitySecret = old })
	req, err := http.NewRequest("POST", "https://cpa.example/v1/tools/web/search", nil)
	require.NoError(t, err)
	info := &relaycommon.RelayInfo{UserId: 42, IsPlayground: true, RequestId: "req-pg"}
	require.NoError(t, ApplyManagedWebAudit(req, info))
	require.Equal(t, "2", req.Header.Get(cpaAuditHeaderVersion))
	require.Equal(t, "playground", req.Header.Get(cpaAuditHeaderIdentityKind))
	require.Equal(t, "42", req.Header.Get(cpaAuditHeaderUserID))
	require.Equal(t, "0", req.Header.Get(cpaAuditHeaderTokenID))
	require.NotEmpty(t, req.Header.Get(cpaAuditHeaderSignature))
}
