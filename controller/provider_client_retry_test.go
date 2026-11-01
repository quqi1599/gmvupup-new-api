package controller

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	"github.com/QuantumNous/new-api/dto"
	"github.com/QuantumNous/new-api/middleware"
	"github.com/QuantumNous/new-api/model"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	"github.com/QuantumNous/new-api/service"
	"github.com/QuantumNous/new-api/setting/ratio_setting"
	"github.com/QuantumNous/new-api/types"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

const providerClientRejection = `{"error":{"type":"invalid_request_error","message":"Illegal API invocation from an unapproved channel"}}`

func TestProviderClientRejectionFallback(t *testing.T) {
	for _, chat := range []bool{false, true} {
		for _, stream := range []bool{false, true} {
			for _, cached := range []bool{false, true} {
				t.Run(fmt.Sprintf("chat=%t/stream=%t/cache=%t", chat, stream, cached), func(t *testing.T) {
					got := runCPAStateRelay(t, "", 400, true, stream, cpaStateAliasOptions{
						chat: chat, cached: cached, maxAttempts: 4,
						ownerHandler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
							w.Header().Set("Content-Type", "application/json")
							w.WriteHeader(400)
							_, _ = io.WriteString(w, providerClientRejection)
						}),
					})
					require.EqualValues(t, 1, got.ownerCalls, "same URL/key alias must not repeat the rejection")
					require.EqualValues(t, 1, got.alternateCalls)
					require.Equal(t, 200, got.response.Code, got.response.Body.String())
					require.NotContains(t, got.response.Body.String(), "unapproved")
					require.Equal(t, service.RetryStopReasonSuccess, got.context.GetString("retry_stop_reason"))
					if stream {
						require.Contains(t, got.response.Body.String(), "data:")
					}
				})
			}
		}
	}
}

func TestProviderClientRejectionStopsWithoutSafeAlternate(t *testing.T) {
	for _, tc := range []struct {
		name, body, response, stopReason string
		alternate, trustedCPA            bool
		headers                          http.Header
	}{
		{name: "no_alternate", response: providerClientRejection, stopReason: service.RetryStopReasonNoChannel},
		{name: "ordinary_parameter_error", response: `{"error":{"type":"invalid_request_error","message":"invalid tools"}}`, alternate: true, stopReason: service.RetryStopReasonNotRetryable},
		{name: "wire_code_not_authority", response: `{"error":{"code":"upstream_client_rejected","message":"ordinary invalid request"}}`, alternate: true, stopReason: service.RetryStopReasonNotRetryable},
		{name: "opaque_history", body: `{"model":"gpt-5.5","previous_response_id":"resp_owned","input":"synthetic"}`, response: providerClientRejection, alternate: true, stopReason: service.RetryStopReasonUpstreamUnknown},
		{name: "content_policy", response: `{"error":{"type":"content_policy_violation","code":"content_policy_violation","message":"Illegal API invocation from an unapproved channel"}}`, alternate: true, stopReason: service.RetryStopReasonNotRetryable},
		{name: "trusted_replay_barrier", response: providerClientRejection, alternate: true, trustedCPA: true, headers: http.Header{"X-Cpa-Retry-Scope": {"request"}, "X-Cpa-Retry-Reason": {service.RetryStopReasonAcceptedOutcomeUnknown}, "X-Cpa-Retry-Attempts": {"1"}}, stopReason: service.RetryStopReasonAcceptedOutcomeUnknown},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got := runCPAStateRelay(t, "", 400, tc.alternate, false, cpaStateAliasOptions{
				maxAttempts: 4, requestBody: tc.body, trustedCPA: tc.trustedCPA,
				ownerHandler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					for key, values := range tc.headers {
						w.Header()[key] = values
					}
					w.WriteHeader(400)
					_, _ = io.WriteString(w, tc.response)
				}),
			})
			require.EqualValues(t, 1, got.ownerCalls)
			require.Zero(t, got.alternateCalls)
			require.Equal(t, 400, got.response.Code)
			require.Equal(t, tc.stopReason, got.context.GetString("retry_stop_reason"))
			if tc.response == providerClientRejection {
				require.Contains(t, got.response.Body.String(), `"code":"upstream_client_rejected"`)
			}
		})
	}
}

func TestProviderClientRejectionClaudeFallback(t *testing.T) {
	for _, stream := range []bool{false, true} {
		t.Run(fmt.Sprint(stream), func(t *testing.T) {
			got := runCPAStateRelay(t, "", 400, true, stream, cpaStateAliasOptions{
				claude: true, noAlias: true, maxAttempts: 3,
				ownerHandler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					w.Header().Set("Content-Type", "application/json")
					w.WriteHeader(400)
					_, _ = io.WriteString(w, providerClientRejection)
				}),
			})
			require.EqualValues(t, 1, got.ownerCalls)
			require.EqualValues(t, 1, got.alternateCalls)
			require.Equal(t, 200, got.response.Code, got.response.Body.String())
			require.Contains(t, got.response.Body.String(), "ok")
			require.NotContains(t, got.response.Body.String(), "unapproved")
			if stream {
				require.Contains(t, got.response.Body.String(), "message_stop")
			}
		})
	}
}

func TestProviderClientRejectionRequestSafety(t *testing.T) {
	for _, tc := range []struct {
		body string
		safe bool
	}{
		{`{"input":"hello","tools":[{"type":"web_search"}]}`, true},
		{`{"messages":[{"role":"user","content":"hello"}],"temperature":0,"stream":true}`, true},
		{`{"input":"hello","tools":[{"type":"function","name":"lookup","parameters":{"properties":{"signature":{"type":"string"}}}}],"metadata":{"file_id":"user-label"},"text":{"format":{"schema":{"properties":{"type":{"const":"compaction"}}}}}}`, true},
		{`{"input":"hello","previous_response_id":"resp_owned"}`, false},
		{`{"input":"hello","conversation":{"id":"conv_owned"}}`, false},
		{`{"input":[{"type":"reasoning","encrypted_content":"opaque"}]}`, false},
		{`{"messages":[{"role":"assistant","content":[{"type":"thinking","thinking":"x","signature":"opaque"}]}]}`, false},
		{`{"input":[{"type":"item_reference","id":"msg_owned"}]}`, false},
		{`{"input":[{"type":"message","content":[{"type":"input_file","file_id":"file_owned"}]}]}`, false},
		{`{"input":"hello","tools":[{"type":"file_search","vector_store_ids":["vs_owned"]}]}`, false},
		{`{"input":"hello","tools":[{"type":"code_interpreter","container":"cntr_owned"}]}`, false},
		{`{"input":"hello","prompt":{"id":"pmpt_owned"}}`, false},
		{`{"input":"hello","background":true}`, false},
		{`["hello"]`, false},
		{`invalid`, false},
	} {
		for _, format := range []types.RelayFormat{types.RelayFormatOpenAI, types.RelayFormatOpenAIResponses, types.RelayFormatClaude} {
			c, _ := gin.CreateTestContext(httptest.NewRecorder())
			c.Request = httptest.NewRequest(http.MethodPost, "/v1/responses", strings.NewReader(tc.body))
			info := &relaycommon.RelayInfo{ChannelMeta: &relaycommon.ChannelMeta{}}
			err := parseCPAContractError(t, 400, providerClientRejection)
			require.Equal(t, !tc.safe, relayRetryIsUnsafe(c, info, format, err), tc.body)
		}
	}
}

func TestProviderClientRejectionRespectsReplayAndSelectionGuards(t *testing.T) {
	for _, guard := range []string{"output", "event", "skip_retry", "specific_channel", "no_budget", "cancelled", "param_override", "managed_web", "gateway", "image"} {
		t.Run(guard, func(t *testing.T) {
			c, _ := gin.CreateTestContext(httptest.NewRecorder())
			c.Request = httptest.NewRequest(http.MethodPost, "/v1/chat/completions", strings.NewReader(`{"messages":[{"role":"user","content":"hello"}]}`))
			info := &relaycommon.RelayInfo{ChannelMeta: &relaycommon.ChannelMeta{}}
			err := parseCPAContractError(t, 400, providerClientRejection)
			format, budget := types.RelayFormat(types.RelayFormatOpenAI), 3
			switch guard {
			case "output":
				_, _ = c.Writer.WriteString("started")
			case "event":
				info.ReceivedResponseCount = 1
			case "skip_retry":
				err = types.NewError(err, err.GetErrorCode(), types.ErrOptionWithSkipRetry())
			case "specific_channel":
				c.Set("specific_channel_id", 147)
			case "no_budget":
				budget = 0
			case "cancelled":
				ctx, cancel := context.WithCancel(c.Request.Context())
				cancel()
				c.Request = c.Request.WithContext(ctx)
			case "param_override":
				info.ParamOverride = map[string]interface{}{"previous_response_id": "resp_owned"}
			case "managed_web":
				info.ManagedWeb = &relaycommon.ManagedWebBilling{SearchRequests: 1}
			case "gateway":
				common.SetContextKey(c, constant.ContextKeyRelayGatewayNoReplayReason, service.RetryStopReasonUpstreamOutputObserved)
			case "image":
				format = types.RelayFormatOpenAIImage
			}
			require.False(t, shouldRetry(c, err, budget, format) && !relayRetryIsUnsafe(c, info, format, err))
		})
	}
}

// Use real wallet/token reservations and the complete HTTP relay so one
// rejected attempt followed by success can be checked against the ledger.
func TestProviderClientRejectionPreservesRequestAndChargesOnce(t *testing.T) {
	setupCPAContractChannels(t, true)
	require.NoError(t, model.DB.AutoMigrate(&model.Token{}, &model.Log{}))
	oldRetry, oldConsume, oldBatch, oldError := common.RetryTimes, common.LogConsumeEnabled, common.BatchUpdateEnabled, constant.ErrorLogEnabled
	oldPrice, oldGroup := ratio_setting.ModelPrice2JSONString(), ratio_setting.GroupRatio2JSONString()
	t.Cleanup(func() {
		common.RetryTimes, common.LogConsumeEnabled, common.BatchUpdateEnabled, constant.ErrorLogEnabled = oldRetry, oldConsume, oldBatch, oldError
		require.NoError(t, ratio_setting.UpdateModelPriceByJSONString(oldPrice))
		require.NoError(t, ratio_setting.UpdateGroupRatioByJSONString(oldGroup))
	})
	common.RetryTimes, common.LogConsumeEnabled, common.BatchUpdateEnabled, constant.ErrorLogEnabled = 3, true, false, false
	require.NoError(t, ratio_setting.UpdateGroupRatioByJSONString(`{"default":1}`))
	require.NoError(t, ratio_setting.UpdateModelPriceByJSONString(`{"deepseek-v4.1-flash":0.01}`))
	t.Setenv("RELAY_RETRY_MAX_ATTEMPTS", "4")
	t.Setenv("RELAY_RETRY_ROUND_BACKOFF_MS", "0")
	service.InitHttpClient()
	const initialQuota = 100000
	user := model.User{Username: "provider-rejection-fixture", Status: common.UserStatusEnabled, Quota: initialQuota}
	require.NoError(t, model.DB.Create(&user).Error)
	token := model.Token{UserId: user.Id, Key: "synthetic-provider-rejection-key", Name: "fixture", Status: common.TokenStatusEnabled, RemainQuota: initialQuota, ExpiredTime: -1}
	require.NoError(t, model.DB.Create(&token).Error)
	const requestBody = `{"model":"deepseek-v4.1-flash","messages":[{"role":"user","content":"synthetic exact content"}],"temperature":0,"top_p":0,"tools":[{"type":"function","function":{"name":"lookup","parameters":{"type":"object","properties":{"signature":{"type":"string"}}}}}],"response_format":{"type":"json_object"},"stream":false}`
	var rejected, accepted atomic.Int32
	bodies := make(chan string, 2)
	var first model.Channel
	for i := 0; i < 2; i++ {
		isFirst := i == 0
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			body, readErr := io.ReadAll(r.Body)
			if readErr != nil {
				t.Errorf("read request: %v", readErr)
			}
			bodies <- string(body)
			w.Header().Set("Content-Type", "application/json")
			if isFirst {
				rejected.Add(1)
				w.WriteHeader(400)
				_, _ = io.WriteString(w, providerClientRejection)
				return
			}
			accepted.Add(1)
			_, _ = io.WriteString(w, `{"id":"chatcmpl_fixture","object":"chat.completion","model":"deepseek-v4.1-flash","choices":[{"index":0,"message":{"role":"assistant","content":"{}"},"finish_reason":"stop"}],"usage":{"prompt_tokens":10,"completion_tokens":2,"total_tokens":12}}`)
		}))
		t.Cleanup(server.Close)
		priority := int64(100 - i*10)
		ch := model.Channel{Id: 9 + i, Type: constant.ChannelTypeOpenAI, Status: common.ChannelStatusEnabled, Group: "default", Models: "deepseek-v4.1-flash", BaseURL: &server.URL, Key: "synthetic-key", Priority: &priority, AutoBan: common.GetPointer(0)}
		require.NoError(t, model.DB.Save(&ch).Error)
		require.NoError(t, model.DB.Model(&model.Ability{}).Where("channel_id = ?", ch.Id).Updates(map[string]interface{}{"priority": priority, "model": "deepseek-v4.1-flash"}).Error)
		if isFirst {
			first = ch
		}
	}
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodPost, "/v1/chat/completions", strings.NewReader(requestBody))
	c.Request.Header.Set("Content-Type", "application/json")
	common.SetContextKey(c, constant.ContextKeyUserId, user.Id)
	common.SetContextKey(c, constant.ContextKeyUserQuota, initialQuota)
	common.SetContextKey(c, constant.ContextKeyTokenId, token.Id)
	common.SetContextKey(c, constant.ContextKeyTokenKey, token.Key)
	common.SetContextKey(c, constant.ContextKeyUserSetting, dto.UserSetting{BillingPreference: "wallet_only"})
	for _, key := range []constant.ContextKey{constant.ContextKeyUsingGroup, constant.ContextKeyUserGroup, constant.ContextKeyTokenGroup} {
		common.SetContextKey(c, key, "default")
	}
	require.Nil(t, middleware.SetupContextForSelectedChannel(c, &first, "deepseek-v4.1-flash"))
	Relay(c, types.RelayFormatOpenAI)
	require.Equal(t, 200, w.Code, w.Body.String())
	require.EqualValues(t, 1, rejected.Load())
	require.EqualValues(t, 1, accepted.Load())
	for i := 0; i < 2; i++ {
		select {
		case body := <-bodies:
			// DTO normalization may add defaults, but customer data and explicit
			// zero values must survive identically on both configured providers.
			var got, original map[string]interface{}
			require.NoError(t, common.Unmarshal([]byte(body), &got))
			require.NoError(t, common.Unmarshal([]byte(requestBody), &original))
			for key, want := range original {
				require.Equal(t, want, got[key], "field=%s attempt=%d", key, i)
			}
		case <-time.After(time.Second):
			t.Fatal("missing upstream request")
		}
	}
	var logs []model.Log
	require.NoError(t, model.LOG_DB.Where("type = ?", model.LogTypeConsume).Find(&logs).Error)
	require.Len(t, logs, 1)
	wantQuota := common.QuotaFromFloat(0.01 * common.QuotaPerUnit)
	require.Equal(t, wantQuota, logs[0].Quota)
	require.Equal(t, 10, logs[0].ChannelId)
	require.NoError(t, model.DB.First(&user, user.Id).Error)
	require.NoError(t, model.DB.First(&token, token.Id).Error)
	require.Equal(t, initialQuota-wantQuota, user.Quota)
	require.Equal(t, initialQuota-wantQuota, token.RemainQuota)
}
