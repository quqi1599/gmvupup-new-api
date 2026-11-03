package controller

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strconv"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	"github.com/QuantumNous/new-api/dto"
	"github.com/QuantumNous/new-api/i18n"
	"github.com/QuantumNous/new-api/middleware"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/service"
	"github.com/QuantumNous/new-api/setting/ratio_setting"
	"github.com/QuantumNous/new-api/setting/system_setting"
	"github.com/alicebob/miniredis/v2"
	"github.com/bytedance/gopkg/util/gopool"
	"github.com/gin-contrib/sessions"
	"github.com/gin-contrib/sessions/cookie"
	"github.com/gin-gonic/gin"
	"github.com/go-redis/redis/v8"
	"github.com/stretchr/testify/require"
)

// These requests enter through the same session authentication, distributor and
// Playground controller as the browser, rather than a hand-built RelayInfo.
// Only the model provider is replaced with a local HTTP fixture.
func TestPlaygroundCPASignedSessionAndWallet(t *testing.T) {
	for _, scenario := range []struct {
		name                          string
		login, mismatch, stream, fail bool
		unconfigured, anthropic       bool
	}{
		{name: "anonymous"},
		{name: "wrong_user_header", login: true, mismatch: true},
		{name: "signed_json", login: true},
		{name: "signed_stream", login: true, stream: true},
		{name: "signed_anthropic_json", login: true, anthropic: true},
		{name: "signed_anthropic_stream", login: true, anthropic: true, stream: true},
		{name: "upstream_rejection_refunds_wallet", login: true, fail: true},
		{name: "unconfigured_authority_refunds_wallet", login: true, unconfigured: true},
	} {
		t.Run(scenario.name, func(t *testing.T) {
			setupCPAContractChannels(t, false)
			require.NoError(t, model.DB.AutoMigrate(&model.Token{}, &model.Log{}))
			require.NoError(t, i18n.Init())
			db, err := model.DB.DB()
			require.NoError(t, err)
			db.SetMaxOpenConns(1)
			oldSecret, oldConsume, oldError := common.CPAAuditIdentitySecret, common.LogConsumeEnabled, constant.ErrorLogEnabled
			oldBatch, oldExport, oldRetry := common.BatchUpdateEnabled, common.DataExportEnabled, common.RetryTimes
			oldGroup, oldRatio, oldCompletion := ratio_setting.GroupRatio2JSONString(), ratio_setting.ModelRatio2JSONString(), ratio_setting.CompletionRatio2JSONString()
			oldRedis, oldRDB := common.RedisEnabled, common.RDB
			oldAuthority := system_setting.ServerAddress
			system_setting.ServerAddress = "https://synthetic-newapi.test"
			if scenario.unconfigured {
				system_setting.ServerAddress = "http://localhost:3000"
			}
			common.CPAAuditIdentitySecret, common.LogConsumeEnabled, constant.ErrorLogEnabled = "playground-fixture-secret", true, false
			common.BatchUpdateEnabled, common.DataExportEnabled, common.RetryTimes = false, false, 0
			circuit := miniredis.RunT(t)
			client := redis.NewClient(&redis.Options{Addr: circuit.Addr()})
			common.RedisEnabled, common.RDB = true, client
			t.Cleanup(func() {
				// Quota-cache updates can finish after the wallet transaction; drain
				// them before restoring global clients and closing this fixture DB.
				require.Eventually(t, func() bool { return gopool.WorkerCount() == 0 }, 3*time.Second, 10*time.Millisecond)
				system_setting.ServerAddress = oldAuthority
				common.CPAAuditIdentitySecret, common.LogConsumeEnabled, constant.ErrorLogEnabled = oldSecret, oldConsume, oldError
				common.BatchUpdateEnabled, common.DataExportEnabled, common.RetryTimes = oldBatch, oldExport, oldRetry
				common.RedisEnabled, common.RDB = oldRedis, oldRDB
				_ = client.Close()
				require.NoError(t, ratio_setting.UpdateGroupRatioByJSONString(oldGroup))
				require.NoError(t, ratio_setting.UpdateModelRatioByJSONString(oldRatio))
				require.NoError(t, ratio_setting.UpdateCompletionRatioByJSONString(oldCompletion))
			})
			require.NoError(t, ratio_setting.UpdateGroupRatioByJSONString(`{"default":1}`))
			require.NoError(t, ratio_setting.UpdateModelRatioByJSONString(`{"pg-identity-fixture":1}`))
			require.NoError(t, ratio_setting.UpdateCompletionRatioByJSONString(`{"pg-identity-fixture":1}`))
			t.Setenv("RELAY_RETRY_MAX_ATTEMPTS", "1")
			t.Setenv("RELAY_RETRY_MAX_ROUNDS", "1")
			service.InitHttpClient()

			const userID, initialQuota, realTokenQuota = 42, 10_000_000, 900_000
			user := model.User{Id: userID, Username: "playground-fixture", Role: common.RoleCommonUser, Status: common.UserStatusEnabled,
				Group: "default", Quota: initialQuota, Setting: `{"billing_preference":"wallet_only"}`}
			require.NoError(t, model.DB.Create(&user).Error)
			realToken := model.Token{Id: 71, UserId: userID, Key: "unused-fixture-token", Status: common.TokenStatusEnabled, RemainQuota: realTokenQuota}
			require.NoError(t, model.DB.Create(&realToken).Error)
			var calls atomic.Int32
			observed := make(chan error, 1)
			expectedPath := "/v1/chat/completions"
			if scenario.anthropic {
				expectedPath = "/v1/messages"
			}
			upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				calls.Add(1)
				var current model.User
				checkErr := model.DB.First(&current, userID).Error
				if checkErr == nil && current.Quota >= initialQuota {
					checkErr = fmt.Errorf("wallet was not reserved before dispatch: %d", current.Quota)
				}
				if checkErr == nil {
					checkErr = validatePlaygroundFixtureSignature(r, common.CPAAuditIdentitySecret, userID, expectedPath)
				}
				observed <- checkErr
				w.Header().Set("Content-Type", "application/json")
				if checkErr != nil {
					w.WriteHeader(http.StatusUnauthorized)
					_, _ = fmt.Fprint(w, `{"error":{"code":"invalid_fixture_identity","message":"fixture identity rejected"}}`)
					return
				}
				if scenario.fail {
					w.WriteHeader(http.StatusBadRequest)
					_, _ = fmt.Fprint(w, `{"error":{"code":"invalid_request_error","message":"synthetic rejection"}}`)
					return
				}

				if scenario.anthropic {
					if scenario.stream {
						w.Header().Set("Content-Type", "text/event-stream")
						_, _ = fmt.Fprint(w, `event: message_start
data: {"type":"message_start","message":{"id":"msg_pg","type":"message","role":"assistant","model":"pg-identity-fixture","content":[],"usage":{"input_tokens":100,"output_tokens":0}}}

event: content_block_start
data: {"type":"content_block_start","index":0,"content_block":{"type":"text","text":""}}

event: content_block_delta
data: {"type":"content_block_delta","index":0,"delta":{"type":"text_delta","text":"fixture-ok"}}

event: content_block_stop
data: {"type":"content_block_stop","index":0}

event: message_delta
data: {"type":"message_delta","delta":{"stop_reason":"end_turn","stop_sequence":null},"usage":{"output_tokens":20}}

event: message_stop
data: {"type":"message_stop"}

`)
					} else {
						_, _ = fmt.Fprint(w, `{"id":"msg_pg","type":"message","role":"assistant","model":"pg-identity-fixture","content":[{"type":"text","text":"fixture-ok"}],"stop_reason":"end_turn","usage":{"input_tokens":100,"output_tokens":20}}`)
					}
					return
				}
				if scenario.stream {
					w.Header().Set("Content-Type", "text/event-stream")
					_, _ = fmt.Fprint(w, "data: {\"id\":\"chatcmpl_pg\",\"object\":\"chat.completion.chunk\",\"model\":\"pg-identity-fixture\",\"choices\":[{\"index\":0,\"delta\":{\"role\":\"assistant\",\"content\":\"fixture-ok\"},\"finish_reason\":null}]}\n\ndata: {\"id\":\"chatcmpl_pg\",\"object\":\"chat.completion.chunk\",\"model\":\"pg-identity-fixture\",\"choices\":[{\"index\":0,\"delta\":{},\"finish_reason\":\"stop\"}],\"usage\":{\"prompt_tokens\":100,\"completion_tokens\":20,\"total_tokens\":120}}\n\ndata: [DONE]\n\n")
					return
				}
				_, _ = fmt.Fprint(w, `{"id":"chatcmpl_pg","object":"chat.completion","model":"pg-identity-fixture","choices":[{"index":0,"message":{"role":"assistant","content":"fixture-ok"},"finish_reason":"stop"}],"usage":{"prompt_tokens":100,"completion_tokens":20,"total_tokens":120}}`)
			}))
			defer upstream.Close()
			channel := model.Channel{Id: 9, Type: constant.ChannelTypeOpenAI, Status: common.ChannelStatusEnabled, Group: "default", Models: "pg-identity-fixture", BaseURL: &upstream.URL, Key: "synthetic-cpa-key"}
			if scenario.anthropic {
				channel.Type = constant.ChannelTypeAnthropic
			}
			channel.SetOtherSettings(dto.ChannelOtherSettings{CPAAuditIdentityEnabled: true})
			require.NoError(t, model.DB.Save(&channel).Error)
			require.NoError(t, model.DB.Model(&model.Ability{}).Where("channel_id = ?", 9).Update("model", "pg-identity-fixture").Error)

			router := gin.New()
			router.Use(sessions.Sessions("pg-fixture", cookie.NewStore([]byte("fixture-only-session-secret"))), middleware.RequestId())
			router.GET("/fixture-login", func(c *gin.Context) {
				session := sessions.Default(c)
				session.Set("username", user.Username)
				session.Set("id", userID)
				session.Set("role", common.RoleCommonUser)
				session.Set("status", common.UserStatusEnabled)
				session.Set("group", "default")
				require.NoError(t, session.Save())
				c.Status(http.StatusNoContent)
			})
			router.POST("/pg/chat/completions", middleware.UserAuth(), middleware.Distribute(), Playground)
			request := httptest.NewRequest(http.MethodPost, "/pg/chat/completions", strings.NewReader(fmt.Sprintf(`{"model":"pg-identity-fixture","group":"default","messages":[{"role":"user","content":"synthetic fixture"}],"max_tokens":32,"stream":%t}`, scenario.stream)))
			request.Header.Set("Content-Type", "application/json")
			request.Header.Set("New-Api-User", strconv.Itoa(userID))
			request.Header.Set(common.RequestIdKey, "pg-fixture-request")
			// A logged-in browser cannot impersonate another user's signed claim.
			request.Header.Set("X-CPA-Audit-User-ID", "999")
			request.Header.Set("X-CPA-Audit-Token-ID", "999")
			request.Header.Set("X-CPA-Audit-Signature", "client-forged")
			request.Header.Set("X-CPA-Audit-Identity-Kind", "channel_test")
			if scenario.login {
				login := httptest.NewRecorder()
				router.ServeHTTP(login, httptest.NewRequest(http.MethodGet, "/fixture-login", nil))
				for _, c := range login.Result().Cookies() {
					request.AddCookie(c)
				}
			}
			if scenario.mismatch {
				request.Header.Set("New-Api-User", "999")
			}
			recorder := httptest.NewRecorder()
			router.ServeHTTP(recorder, request)
			expectedQuota, expectedConsumes := initialQuota, int64(0)
			if !scenario.login || scenario.mismatch {
				require.Equal(t, http.StatusUnauthorized, recorder.Code, recorder.Body.String())
				require.Zero(t, calls.Load())
			} else if scenario.unconfigured {
				require.Equal(t, http.StatusInternalServerError, recorder.Code)
				require.Contains(t, recorder.Body.String(), "channel:cpa_audit_identity_invalid")
				require.Zero(t, calls.Load())
			} else {
				require.EqualValues(t, 1, calls.Load(), recorder.Body.String())
				require.NoError(t, <-observed)
				if scenario.fail {
					require.Equal(t, http.StatusBadRequest, recorder.Code, recorder.Body.String())
				} else {
					require.Equal(t, http.StatusOK, recorder.Code, recorder.Body.String())
					require.Contains(t, recorder.Body.String(), "fixture-ok")
					if scenario.stream {
						require.Contains(t, recorder.Body.String(), "[DONE]")
					}
					expectedQuota, expectedConsumes = initialQuota-120, 1
				}
			}
			// Refund runs asynchronously. Wait for the observable wallet/log state,
			// not for an arbitrary sleep or an HTTP header alone.
			require.Eventually(t, func() bool {
				var current model.User
				var count int64
				return model.DB.First(&current, userID).Error == nil && current.Quota == expectedQuota &&
					model.LOG_DB.Model(&model.Log{}).Where("request_id = ? AND type = ?", "pg-fixture-request", model.LogTypeConsume).Count(&count).Error == nil && count == expectedConsumes
			}, 3*time.Second, 10*time.Millisecond)
			var tokens []model.Token
			require.NoError(t, model.DB.Find(&tokens).Error)
			require.Len(t, tokens, 1)
			require.Equal(t, realTokenQuota, tokens[0].RemainQuota)
			require.Zero(t, tokens[0].UsedQuota)
			if expectedConsumes > 0 {
				var log model.Log
				require.NoError(t, model.LOG_DB.Where("request_id = ? AND type = ?", "pg-fixture-request", model.LogTypeConsume).First(&log).Error)
				require.Equal(t, userID, log.UserId)
				require.Zero(t, log.TokenId)
				require.Equal(t, 120, log.Quota)
			}
		})
	}
}

// Validate the wire contract independently of the production signing helper.
func validatePlaygroundFixtureSignature(r *http.Request, secret string, userID int, expectedPath string) error {
	h := r.Header
	if r.Method != http.MethodPost || r.URL.Path != expectedPath ||
		h.Get("X-CPA-Audit-Version") != "2" || h.Get("X-CPA-Audit-Identity-Kind") != "playground" ||
		h.Get("X-CPA-Audit-User-ID") != strconv.Itoa(userID) || h.Get("X-CPA-Audit-Token-ID") != "0" ||
		h.Get("X-CPA-Audit-Token-Name") != "__playground__" || h.Get("X-CPA-Audit-Channel-Test") == "1" ||
		h.Get("X-CPA-Audit-Request-ID") != "pg-fixture-request" {
		return fmt.Errorf("unexpected signed playground identity")
	}
	values := url.Values{"version": {"2"}, "identity_kind": {"playground"}, "user_id": {strconv.Itoa(userID)},
		"token_id": {"0"}, "token_name": {"__playground__"}, "channel_test": {"0"},
		"method": {r.Method}, "path": {r.URL.EscapedPath()}, "request_id": {"pg-fixture-request"},
		"timestamp": {h.Get("X-CPA-Audit-Timestamp")}, "model": {h.Get("X-CPA-Audit-Model")}}
	if h.Get("X-CPA-Audit-Signature") != common.HmacSha256(values.Encode(), secret) {
		return fmt.Errorf("playground HMAC did not bind the identity and route")
	}
	expectedPrincipal := common.HmacSha256("newapi/cpa-principal/playground/v1\x00https://synthetic-newapi.test\x00"+strconv.Itoa(userID), "synthetic-cpa-key")
	if h.Get("X-CPA-Principal") != expectedPrincipal {
		return fmt.Errorf("playground principal did not isolate the logged-in user")
	}
	return nil
}
