package router

import (
	"bytes"
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
	"github.com/QuantumNous/new-api/setting/ratio_setting"
	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	"github.com/stretchr/testify/require"
	"github.com/tidwall/gjson"
)

func TestCodexResponsesWebSocketToolContinuation(t *testing.T) {
	user, token := setupJevRelayTest(t)
	oldRatios := ratio_setting.ModelRatio2JSONString()
	require.NoError(t, ratio_setting.UpdateModelRatioByJSONString(`{"gpt-4o-mini":1}`))
	t.Cleanup(func() { require.NoError(t, ratio_setting.UpdateModelRatioByJSONString(oldRatios)) })
	oldLogs := common.LogConsumeEnabled
	common.LogConsumeEnabled = true
	t.Cleanup(func() { common.LogConsumeEnabled = oldLogs })
	var sockets, creates, posts atomic.Int32
	var up websocket.Upgrader
	backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/responses" || r.Header.Get("Authorization") != "Bearer synthetic-provider" {
			http.Error(w, "wrong upstream endpoint or credential", 400)
			return
		}
		if r.Method == http.MethodPost {
			b, _ := io.ReadAll(r.Body)
			items := gjson.GetBytes(b, "input").Array()
			if len(items) != 2 || items[0].Get("role").Str != "user" || !strings.Contains(items[0].Get("content.0.text").Str, "fixture tab") || items[1].Get("call_id").Str != "call_http" {
				http.Error(w, "function_call_output requires call_id", 400)
				return
			}
			posts.Add(1)
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprint(w, `{"id":"resp_http","object":"response","status":"completed","output":[],"usage":{"input_tokens":10,"output_tokens":2,"total_tokens":12}}`)
			return
		}
		if !websocket.IsWebSocketUpgrade(r) || r.Header.Get("OpenAI-Beta") != "responses_websockets=2026-02-06" {
			http.Error(w, "expected v2", 400)
			return
		}
		c, err := up.Upgrade(w, r, nil)
		if err != nil {
			return
		}
		defer c.Close()
		sockets.Add(1)
		for round := 0; ; round++ {
			_, data, err := c.ReadMessage()
			if err != nil {
				return
			}
			creates.Add(1)
			obj := gjson.ParseBytes(data)
			if obj.Get("type").Str != "response.create" || obj.Get("stream").Exists() {
				t.Errorf("invalid upstream envelope")
				return
			}
			var id, output string
			input, outputTokens := 10, 2
			switch round {
			case 0:
				if g := obj.Get("generate"); !g.Exists() || g.Bool() {
					t.Errorf("warmup generated")
					return
				}
				id, output, input, outputTokens = "resp_warm", "[]", 0, 0
			case 1:
				if obj.Get("previous_response_id").Str != "resp_warm" || obj.Get("input.0.role").Str != "user" {
					t.Errorf("lost normalized context or warmup ID")
					return
				}
				id, output = "resp_tool", `[{"type":"function_call","id":"fc_test","call_id":"call_actual","name":"lookup","arguments":"{}","status":"completed"}]`
			case 2:
				items := obj.Get("input").Array()
				if obj.Get("previous_response_id").Str != "resp_tool" || len(items) != 1 || items[0].Get("call_id").Str != "call_actual" || items[0].Get("output").Str != "杭州 23℃" {
					t.Errorf("tool result did not continue the original response")
					return
				}
				id, output = "resp_answer", `[{"type":"message","role":"assistant","content":[{"type":"output_text","text":"杭州 23℃"}]}]`
			case 3:
				if err := c.WriteMessage(websocket.TextMessage, []byte(`{"type":"response.created","response":{"id":"resp_cancel","status":"in_progress"}}`)); err != nil {
					return
				}
				_, raw, err := c.ReadMessage()
				if err != nil {
					return
				}
				if j := gjson.ParseBytes(raw); j.Get("type").Str != "response.cancel" || j.Get("response_id").Str != "resp_cancel" {
					t.Errorf("unexpected control in active generation")
					return
				}
				if err := c.WriteMessage(websocket.TextMessage, []byte(`{"type":"response.cancelled","response":{"id":"resp_cancel","status":"cancelled"}}`)); err != nil {
					return
				}
				continue
			case 4:
				_ = c.WriteMessage(websocket.TextMessage, []byte(`{"type":"response.created","response":{"id":"resp_lost","status":"in_progress"}}`))
				return // transport closes before a terminal event; generation outcome is unknown
			default:
				t.Errorf("unexpected extra upstream turn")
				return
			}
			created := fmt.Sprintf(`{"type":"response.created","response":{"id":%q,"status":"in_progress"}}`, id)
			if round == 2 {
				// A delayed terminal must not settle the next turn or replace its parent.
				_ = c.WriteMessage(websocket.TextMessage, []byte(`{"type":"response.completed","response":{"id":"resp_tool","status":"completed","output":[]}}`))
			}
			if err := c.WriteMessage(websocket.TextMessage, []byte(created)); err != nil {
				return
			}
			final := fmt.Sprintf(`{"type":"response.completed","response":{"id":%q,"object":"response","status":"completed","output":%s,"usage":{"input_tokens":%d,"output_tokens":%d,"total_tokens":%d}}}`, id, output, input, outputTokens, input+outputTokens)
			if err := c.WriteMessage(websocket.TextMessage, []byte(final)); err != nil {
				return
			}
		}
	}))
	defer backend.Close()
	ch := model.Channel{Name: "fixture", Type: constant.ChannelTypeOpenAI, Key: "synthetic-provider", Status: common.ChannelStatusEnabled, Group: "default", Models: "gpt-4o-mini", BaseURL: &backend.URL}
	ch.SetSetting(dto.ChannelSettings{})
	require.NoError(t, model.DB.Create(&ch).Error)
	require.NoError(t, model.DB.Create(&model.Ability{Group: "default", Model: "gpt-4o-mini", ChannelId: ch.Id, Enabled: true}).Error)
	engine := gin.New()
	engine.Use(middleware.RequestId())
	SetRelayRouter(engine)
	gateway := httptest.NewServer(engine)
	defer gateway.Close()
	addr := "ws" + strings.TrimPrefix(gateway.URL, "http") + "/v1/responses"
	auth := http.Header{"Authorization": []string{"Bearer sk-" + token.Key}}
	_, deny, err := websocket.DefaultDialer.Dial(addr, nil)
	require.Error(t, err)
	require.Equal(t, 401, deny.StatusCode)
	deny.Body.Close()
	_, off, err := websocket.DefaultDialer.Dial(addr, auth)
	require.Error(t, err)
	require.Equal(t, 426, off.StatusCode)
	off.Body.Close()
	request, _ := http.NewRequest(http.MethodPost, gateway.URL+"/v1/responses", bytes.NewBufferString(`{"model":"gpt-4o-mini","input":[{"type":"function_call_output","namespace":"agent","name":"browser_info","output":"fixture tab"},{"type":"function_call_output","call_id":"call_http","output":"ok"}]}`))
	request.Header = auth.Clone()
	request.Header.Set("Content-Type", "application/json")
	resp, err := http.DefaultClient.Do(request)
	require.NoError(t, err)
	b, _ := io.ReadAll(resp.Body)
	resp.Body.Close()
	require.Equal(t, 200, resp.StatusCode, string(b))
	require.EqualValues(t, 1, posts.Load())
	ch.SetSetting(dto.ChannelSettings{ResponsesWebSocketEnabled: true})
	require.NoError(t, model.DB.Model(&ch).Update("setting", ch.Setting).Error)
	// A higher-priority HTTP-only provider must never win WebSocket selection.
	priority := int64(100)
	other := model.Channel{Name: "http-only", Type: constant.ChannelTypeOpenAI, Key: "wrong-upstream", Status: common.ChannelStatusEnabled, Group: "default", Models: "gpt-4o-mini,http-only-model", BaseURL: &backend.URL, Priority: &priority}
	require.NoError(t, model.DB.Create(&other).Error)
	for _, m := range []string{"gpt-4o-mini", "http-only-model"} {
		require.NoError(t, model.DB.Create(&model.Ability{Group: "default", Model: m, ChannelId: other.Id, Enabled: true, Priority: &priority}).Error)
	}
	client, _, err := websocket.DefaultDialer.Dial(addr, auth)
	require.NoError(t, err)
	defer client.Close()
	round := func(s string, expected string) gjson.Result {
		t.Helper()
		require.NoError(t, client.WriteMessage(websocket.TextMessage, []byte(s)))
		require.NoError(t, client.SetReadDeadline(time.Now().Add(8*time.Second)))
		for n := 0; n < 8; n++ {
			_, raw, err := client.ReadMessage()
			require.NoError(t, err)
			j := gjson.ParseBytes(raw)
			if j.Get("type").Str == "response.created" {
				continue
			}
			require.Equal(t, expected, j.Get("type").Str, string(raw))
			return j
		}
		t.Fatal("missing terminal event")
		return gjson.Result{}
	}
	offModel := round(`{"type":"response.create","model":"http-only-model","input":"hello"}`, "error")
	require.Equal(t, int64(426), offModel.Get("status").Int())
	warm := round(`{"type":"response.create","model":"gpt-4o-mini","generate":false}`, "response.completed")
	require.Equal(t, "resp_warm", warm.Get("response.id").Str)
	call := round(`{"type":"response.create","model":"gpt-4o-mini","previous_response_id":"resp_warm","input":[{"type":"function_call_output","namespace":"agent","name":"ide_context","output":"fixture tab"}],"tools":[{"type":"function","name":"lookup","parameters":{"type":"object"}}]}`, "response.completed")
	require.Equal(t, "call_actual", call.Get("response.output.0.call_id").Str)
	answer := round(fmt.Sprintf(`{"type":"response.create","model":"gpt-4o-mini","previous_response_id":%q,"input":[{"type":"function_call_output","call_id":%q,"output":"杭州 23℃"}]}`, call.Get("response.id").Str, call.Get("response.output.0.call_id").Str), "response.completed")
	require.Equal(t, "杭州 23℃", answer.Get("response.output.0.content.0.text").Str)
	// Another socket cannot borrow this connection's server-side parent.
	otherClient, _, err := websocket.DefaultDialer.Dial(addr, auth)
	require.NoError(t, err)
	require.NoError(t, otherClient.WriteMessage(websocket.TextMessage, []byte(`{"type":"response.create","model":"gpt-4o-mini","previous_response_id":"resp_answer","input":"next"}`)))
	require.NoError(t, otherClient.SetReadDeadline(time.Now().Add(3*time.Second)))
	_, foreign, err := otherClient.ReadMessage()
	otherClient.Close()
	require.NoError(t, err)
	require.Equal(t, "previous_response_not_found", gjson.GetBytes(foreign, "error.code").Str)
	// A rotated upstream key is rejected before the existing socket is reused.
	require.NoError(t, model.DB.Model(&ch).Update("key", "rotated-provider").Error)
	rotated := round(`{"type":"response.create","model":"gpt-4o-mini","previous_response_id":"resp_answer","input":"next"}`, "error")
	require.Equal(t, int64(409), rotated.Get("status").Int())
	require.NoError(t, model.DB.Model(&ch).Update("key", "synthetic-provider").Error)
	wrong := round(`{"type":"response.create","model":"gpt-4o-mini","previous_response_id":"resp_other","input":"next"}`, "error")
	require.Equal(t, "previous_response_not_found", wrong.Get("error.code").Str)
	ch.SetSetting(dto.ChannelSettings{})
	require.NoError(t, model.DB.Model(&ch).Update("setting", ch.Setting).Error)
	rejected := round(`{"type":"response.create","model":"gpt-4o-mini","previous_response_id":"resp_answer","input":"next"}`, "error")
	require.GreaterOrEqual(t, rejected.Get("status").Int(), int64(400))
	require.EqualValues(t, 1, sockets.Load())
	require.EqualValues(t, 3, creates.Load())
	ch.SetSetting(dto.ChannelSettings{ResponsesWebSocketEnabled: true})
	require.NoError(t, model.DB.Model(&ch).Update("setting", ch.Setting).Error)
	require.NoError(t, client.WriteMessage(websocket.TextMessage, []byte(`{"type":"response.create","model":"gpt-4o-mini","input":"fixture cancellation","stream_id":"lane2"}`)))
	_, raw, err := client.ReadMessage()
	require.NoError(t, err)
	require.Equal(t, "resp_cancel", gjson.GetBytes(raw, "response.id").Str)
	require.Equal(t, "lane2", gjson.GetBytes(raw, "stream_id").Str)
	busy := round(`{"type":"response.create","model":"gpt-4o-mini","input":"unsent second create"}`, "error")
	require.Equal(t, int64(409), busy.Get("status").Int())
	cancelled := round(`{"type":"response.cancel","response_id":"resp_cancel","stream_id":"lane2"}`, "response.cancelled")
	require.Equal(t, "resp_cancel", cancelled.Get("response.id").Str)
	require.Equal(t, "lane2", cancelled.Get("stream_id").Str)
	interrupted := round(`{"type":"response.create","model":"gpt-4o-mini","input":"fixture disconnect"}`, "response.failed")
	require.NotEmpty(t, interrupted.Get("response.error").Raw)
	_, _, err = client.ReadMessage()
	require.Error(t, err)
	require.EqualValues(t, 5, creates.Load())
	require.EqualValues(t, 1, sockets.Load())
	var charges []model.Log
	require.NoError(t, model.LOG_DB.Where("token_id=? AND type=? AND quota>0", token.Id, model.LogTypeConsume).Order("id").Find(&charges).Error)
	require.Len(t, charges, 3, "one HTTP request and two WS generations")
	total := 0
	seen := map[string]bool{}
	for _, l := range charges {
		require.NotEmpty(t, l.RequestId)
		require.False(t, seen[l.RequestId])
		seen[l.RequestId] = true
		total += l.Quota
	}
	var after model.User
	require.NoError(t, model.DB.First(&after, user.Id).Error)
	require.Equal(t, user.Quota-total, after.Quota)
	var tk model.Token
	require.NoError(t, model.DB.First(&tk, token.Id).Error)
	require.Equal(t, token.RemainQuota-total, tk.RemainQuota)
}
