package relay

import (
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	"github.com/QuantumNous/new-api/dto"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/pkg/websearch"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	relayconstant "github.com/QuantumNous/new-api/relay/constant"
	"github.com/QuantumNous/new-api/relay/helper"
	"github.com/QuantumNous/new-api/service"
	"github.com/QuantumNous/new-api/setting/model_setting"
	"github.com/QuantumNous/new-api/setting/operation_setting"
	"github.com/QuantumNous/new-api/types"
	"github.com/alicebob/miniredis/v2"
	"github.com/gin-gonic/gin"
	"github.com/go-redis/redis/v8"
	"github.com/stretchr/testify/require"
	"github.com/tidwall/gjson"
)

func TestNativeCachedSearchRelayKeepsProviderAndBilling(t *testing.T) {
	service.InitTokenEncoders()
	t.Setenv("UNIFIED_WEB_SEARCH_ENABLED", "true")
	t.Setenv("DEEPSEEK_SEARCH_PROXY_URL", "")
	t.Setenv("DEEPSEEK_SEARCH_PROXY_KEY_FILE", "")
	oldRedis := common.RDB
	common.RDB = nil
	t.Cleanup(func() { common.RDB = oldRedis })
	settings := model_setting.GetGlobalSettings()
	oldPass := settings.PassThroughRequestEnabled
	settings.PassThroughRequestEnabled = false
	oldLog, oldExport := common.LogConsumeEnabled, common.DataExportEnabled
	common.LogConsumeEnabled, common.DataExportEnabled = true, false
	t.Cleanup(func() {
		settings.PassThroughRequestEnabled = oldPass
		common.LogConsumeEnabled, common.DataExportEnabled = oldLog, oldExport
	})
	for _, stream := range []bool{false, true} {
		for _, searched := range []bool{false, true} {
			for _, group := range []int{1, 3} {
				t.Run(fmt.Sprintf("stream=%t/search=%t/group=%d", stream, searched, group), func(t *testing.T) {
					user, token := setupMidjourneyBillingTest(t, 1000000, 1000000)
					body := fmt.Sprintf(`{"model":"gpt-6.1-sol","input":"fixture mentions ws_na_ as ordinary text","temperature":0,"stream":%t,"tools":[{"type":"web_search","external_web_access":false,"user_location":{"type":"approximate","city":"Paris"}},{"type":"function","name":"local_read","parameters":{"type":"object"}}],"tool_choice":"auto"}`, stream)
					recorder := httptest.NewRecorder()
					c, _ := gin.CreateTestContext(recorder)
					c.Request = httptest.NewRequest("POST", "/v1/responses", strings.NewReader(body))
					c.Request.Header.Set("Content-Type", "application/json")
					c.Set(common.RequestIdKey, "cached-search-fixture")
					t.Cleanup(func() { common.CleanupBodyStorage(c) })
					request, err := helper.GetAndValidateRequest(c, types.RelayFormatOpenAIResponses)
					require.NoError(t, err)
					info := &relaycommon.RelayInfo{
						UserId: user.Id, TokenId: token.Id, TokenKey: token.Key, Request: request,
						OriginModelName: "gpt-6.1-sol", RequestURLPath: "/v1/responses",
						UsingGroup: "default", TokenGroup: "default", UserGroup: "default", StartTime: time.Now(),
						IsStream: stream, RelayFormat: types.RelayFormatOpenAIResponses, RelayMode: relayconstant.RelayModeResponses,
						ChannelMeta:        &relaycommon.ChannelMeta{},
						ResponsesUsageInfo: &relaycommon.ResponsesUsageInfo{BuiltInTools: map[string]*relaycommon.BuildInToolInfo{dto.BuildInToolWebSearch: {ToolName: dto.BuildInToolWebSearch}}},
						UserSetting:        dto.UserSetting{BillingPreference: "wallet_only", QuotaWarningThreshold: 1},
						PriceData:          types.PriceData{ModelRatio: 1, CompletionRatio: 1, CacheRatio: 1, GroupRatioInfo: types.GroupRatioInfo{GroupRatio: float64(group)}},
					}
					require.Nil(t, service.PreConsumeBilling(c, 100, info))
					output := `{"type":"message","id":"m_cached","status":"completed","role":"assistant","content":[{"type":"output_text","text":"fixture answer","annotations":[]}]}`
					if searched {
						output = `{"type":"web_search_call","id":"ws_provider","status":"completed","action":{"type":"search","query":"fixture"}},` + output
					}
					payload := `{"id":"r_cached","object":"response","model":"gpt-6.1-sol","status":"completed","output":[` + output + `],"usage":{"input_tokens":10,"output_tokens":2}}`
					received := make(chan []byte, 2)
					remote := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
						raw, readErr := io.ReadAll(r.Body)
						if readErr != nil {
							http.Error(w, "fixture read failed", 500)
							return
						}
						received <- raw
						if stream {
							w.Header().Set("Content-Type", "text/event-stream")
							response, _ := websearch.Obj([]byte(payload))
							_ = websearch.Emit(w, "responses", response)
						} else {
							w.Header().Set("Content-Type", "application/json")
							_, _ = io.WriteString(w, payload)
						}
					}))
					defer remote.Close()
					common.SetContextKey(c, constant.ContextKeyChannelType, constant.ChannelTypeOpenAI)
					common.SetContextKey(c, constant.ContextKeyChannelBaseUrl, remote.URL)
					common.SetContextKey(c, constant.ContextKeyChannelKey, "fixture")
					common.SetContextKey(c, constant.ContextKeyOriginalModel, "gpt-6.1-sol")
					modelRuns := 0
					apiErr := ManagedWebHelper(c, info, func() *types.NewAPIError {
						modelRuns++
						return ResponsesHelper(c, info)
					})
					require.Nil(t, apiErr)
					require.Equal(t, 1, modelRuns)
					require.Nil(t, info.ManagedWeb, "no DeepSeek execution or surcharge")
					require.Len(t, received, 1)
					upstream := <-received
					require.Equal(t, "gpt-6.1-sol", gjson.GetBytes(upstream, "model").String())
					require.JSONEq(t, gjson.Get(body, "tools").Raw, gjson.GetBytes(upstream, "tools").Raw)
					require.Equal(t, "false", gjson.GetBytes(upstream, "tools.0.external_web_access").Raw)
					require.Equal(t, "0", gjson.GetBytes(upstream, "temperature").Raw)
					require.Equal(t, stream, gjson.GetBytes(upstream, "stream").Bool())
					require.NotContains(t, string(upstream), websearch.SearchFunction)
					require.Contains(t, recorder.Body.String(), "fixture answer")
					searches, fee := 0, 0
					if searched {
						searches = 1
						fee = int(operation_setting.GetToolPriceForModel(dto.BuildInToolWebSearch, "gpt-6.1-sol") / 1000 * common.QuotaPerUnit * float64(group))
						require.Positive(t, fee)
						require.Contains(t, recorder.Body.String(), "ws_provider")
					}
					require.Equal(t, searches, info.ResponsesUsageInfo.BuiltInTools[dto.BuildInToolWebSearch].CallCount)
					want := 12*group + fee
					wallet, remaining := persistedMidjourneyQuota(t, user.Id, token.Id)
					require.Equal(t, 1000000-want, wallet)
					require.Equal(t, wallet, remaining)
					var logs []model.Log
					require.NoError(t, model.LOG_DB.Where("user_id=? AND type=?", user.Id, model.LogTypeConsume).Find(&logs).Error)
					require.Len(t, logs, 1)
					require.Equal(t, want, logs[0].Quota)
					require.False(t, gjson.Get(logs[0].Other, "managed_web").Exists())
					info.Billing.Refund(c)
					wallet, _ = persistedMidjourneyQuota(t, user.Id, token.Id)
					require.Equal(t, 1000000-want, wallet, "normal settlement must not be refunded or duplicated")
				})
			}
		}
	}
}

func TestNativeCachedSearchRestoresCallerBoundManagedHistory(t *testing.T) {
	t.Setenv("UNIFIED_WEB_SEARCH_ENABLED", "true")
	t.Setenv("DEEPSEEK_SEARCH_PROXY_URL", "")
	server := miniredis.RunT(t)
	client := redis.NewClient(&redis.Options{Addr: server.Addr()})
	oldRedis := common.RDB
	common.RDB = client
	t.Cleanup(func() { common.RDB = oldRedis; client.Close() })
	store := websearch.RedisStore{Scope: "1:10"}
	require.NoError(t, store.Put(t.Context(), "ws_na_fixture", websearch.Record{
		Call:   websearch.Raw(map[string]any{"type": "function_call", "call_id": "past_call", "name": websearch.SearchFunction, "arguments": `{"query":"fixture"}`}),
		Result: websearch.Raw(map[string]any{"results": []any{map[string]string{"url": "https://example.com", "summary": "past source"}}}),
	}))
	for _, token := range []int{10, 20} {
		t.Run(fmt.Sprintf("token=%d", token), func(t *testing.T) {
			body := `{"model":"gpt-6.1-sol","input":[{"type":"web_search_call","id":"ws_na_fixture","status":"completed","action":{"type":"search","query":"fixture"}},{"role":"user","content":"continue"}],"tools":[{"type":"web_search","external_web_access":false}],"stream":true}`
			c, _ := gin.CreateTestContext(httptest.NewRecorder())
			c.Request = httptest.NewRequest("POST", "/v1/responses", strings.NewReader(body))
			c.Request.Header.Set("Content-Type", "application/json")
			t.Cleanup(func() { common.CleanupBodyStorage(c) })
			original, err := helper.GetAndValidateRequest(c, types.RelayFormatOpenAIResponses)
			require.NoError(t, err)
			storage, err := common.GetBodyStorage(c)
			require.NoError(t, err)
			info := &relaycommon.RelayInfo{UserId: 1, TokenId: token, Request: original, RelayFormat: types.RelayFormatOpenAIResponses, RelayMode: relayconstant.RelayModeResponses, IsStream: true}
			calls := 0
			apiErr := ManagedWebHelper(c, info, func() *types.NewAPIError {
				calls++
				encoded, encodeErr := common.Marshal(info.Request)
				require.NoError(t, encodeErr)
				require.NotContains(t, string(encoded), "ws_na_fixture")
				require.Contains(t, string(encoded), "past source")
				require.Equal(t, "false", gjson.GetBytes(encoded, "tools.0.external_web_access").Raw)
				require.True(t, gjson.GetBytes(encoded, "stream").Bool())
				require.Nil(t, info.ManagedWeb)
				return nil
			})
			if token == 10 {
				require.Nil(t, apiErr)
				require.Equal(t, 1, calls)
			} else {
				require.NotNil(t, apiErr)
				require.Equal(t, 409, apiErr.StatusCode)
				require.Zero(t, calls)
			}
			restoredStorage, err := common.GetBodyStorage(c)
			require.NoError(t, err)
			require.Same(t, storage, restoredStorage)
			require.Same(t, original, info.Request)
		})
	}
}
