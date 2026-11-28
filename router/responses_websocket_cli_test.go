package router

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	"github.com/QuantumNous/new-api/dto"
	"github.com/QuantumNous/new-api/middleware"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/service"
	"github.com/QuantumNous/new-api/setting/ratio_setting"
	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	"github.com/stretchr/testify/require"
	"github.com/tidwall/gjson"
)

// Optional interoperability test with the installed Codex binary. All model
// traffic goes to the loopback fixture; the child gets isolated config and keys.
func TestInstalledCodexToolAndSessionResume(t *testing.T) {
	if os.Getenv("TEST_NEWAPI_CODEX_CLI") != "1" {
		t.Skip("set TEST_NEWAPI_CODEX_CLI=1 to run the installed client")
	}
	bin, err := exec.LookPath("codex")
	require.NoError(t, err)
	_, token := setupJevRelayTest(t)
	service.InitTokenEncoders()
	old := ratio_setting.ModelRatio2JSONString()
	require.NoError(t, ratio_setting.UpdateModelRatioByJSONString(`{"gpt-5.6-sol":1}`))
	t.Cleanup(func() { require.NoError(t, ratio_setting.UpdateModelRatioByJSONString(old)) })
	const marker = "LOCAL_WS_TOOL_27"
	var mu sync.Mutex
	connections, calls, toolResponses, resumed := 0, 0, 0, false
	httpResumes := 0
	var upgrade websocket.Upgrader
	up := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost && r.URL.Path == "/v1/responses" {
			body, err := io.ReadAll(r.Body)
			if err != nil {
				http.Error(w, "fixture read failed", 400)
				return
			}
			input := gjson.GetBytes(body, "input").Array()
			var previousTool, follow bool
			for _, v := range input {
				switch v.Get("type").Str {
				case "function_call_output", "custom_tool_call_output":
					if v.Get("call_id").Str == "call_local" && strings.Contains(v.Get("output").Raw, marker) {
						previousTool = true
					}
				case "message":
					if v.Get("role").Str == "user" && strings.Contains(v.Get("content").Raw, "RESUME_HTTP_SESSION") {
						follow = true
					}
				}
			}
			if !previousTool || !follow {
				http.Error(w, "complete tool history required", 400)
				return
			}
			mu.Lock()
			httpResumes++
			mu.Unlock()
			w.Header().Set("Content-Type", "text/event-stream")
			item := cliFixtureText("msg_http", "HTTP_RESUME_CONFIRMED")
			for _, event := range []map[string]any{
				{"type": "response.created", "response": map[string]any{"id": "resp_http_resume", "status": "in_progress"}},
				{"type": "response.output_item.done", "output_index": 0, "item": item},
				{"type": "response.completed", "response": map[string]any{"id": "resp_http_resume", "status": "completed", "output": []any{item}, "usage": map[string]int{"input_tokens": 10, "output_tokens": 2, "total_tokens": 12}}},
			} {
				frame, _ := common.Marshal(event)
				fmt.Fprintf(w, "data: %s\n\n", frame)
			}
			return
		}
		if !websocket.IsWebSocketUpgrade(r) || r.URL.Path != "/v1/responses" {
			http.NotFound(w, r)
			return
		}
		conn, err := upgrade.Upgrade(w, r, nil)
		if err != nil {
			return
		}
		defer conn.Close()
		mu.Lock()
		connections++
		mu.Unlock()
		var advertisedTools []gjson.Result
		for {
			_, msg, err := conn.ReadMessage()
			if err != nil {
				return
			}
			j := gjson.ParseBytes(msg)
			mu.Lock()
			calls++
			index := calls
			mu.Unlock()
			id := fmt.Sprintf("resp_local_%d", index)
			warm := j.Get("generate").Exists() && !j.Get("generate").Bool()
			if tools := j.Get("tools"); tools.Exists() {
				advertisedTools = tools.Array()
			}
			for _, input := range j.Get("input").Array() {
				if input.Get("type").Str == "additional_tools" {
					advertisedTools = append(advertisedTools, input.Get("tools").Array()...)
				}
			}
			var item map[string]any
			if !warm {
				input := j.Get("input")
				gotTool, follow := false, false
				for _, v := range input.Array() {
					if (v.Get("type").Str == "function_call_output" || v.Get("type").Str == "custom_tool_call_output") && v.Get("call_id").Str == "call_local" && strings.Contains(v.Get("output").Raw, marker) {
						gotTool = true
					}
					if v.Get("role").Str == "user" && strings.Contains(v.Get("content").Raw, "RESUME_LOCAL_SESSION") {
						follow = true
					}
				}
				switch {
				case follow:
					if !gotTool {
						t.Errorf("resumed session did not carry the earlier tool result")
						return
					}
					mu.Lock()
					resumed = true
					mu.Unlock()
					item = cliFixtureText("msg_resume", "RESUME_CONFIRMED")
				case gotTool:
					mu.Lock()
					toolResponses++
					mu.Unlock()
					item = cliFixtureText("msg_tool", "TOOL_RESULT_CONFIRMED")
				default:
					item = cliFixtureTool(advertisedTools)
					if item == nil {
						t.Errorf("Codex did not advertise a supported local test tool")
						return
					}
				}
			}
			events := []map[string]any{{"type": "response.created", "response": map[string]any{"id": id, "object": "response", "status": "in_progress", "output": []any{}}}}
			output := []any{}
			if item != nil {
				output = append(output, item)
				events = append(events, map[string]any{"type": "response.output_item.done", "output_index": 0, "item": item})
			}
			usage := map[string]int{"input_tokens": 10, "output_tokens": 2, "total_tokens": 12}
			if warm {
				usage = map[string]int{"input_tokens": 0, "output_tokens": 0, "total_tokens": 0}
			}
			events = append(events, map[string]any{"type": "response.completed", "response": map[string]any{"id": id, "object": "response", "model": "gpt-5.6-sol", "status": "completed", "output": output, "usage": usage}})
			for i, event := range events {
				event["sequence_number"] = i
				payload, _ := common.Marshal(event)
				if conn.WriteMessage(websocket.TextMessage, payload) != nil {
					return
				}
			}
		}
	}))
	defer up.Close()
	ch := model.Channel{Name: "cli", Type: constant.ChannelTypeOpenAI, Key: "synthetic", Status: common.ChannelStatusEnabled, Group: "default", Models: "gpt-5.6-sol", BaseURL: &up.URL}
	ch.SetSetting(dto.ChannelSettings{ResponsesWebSocketEnabled: true})
	require.NoError(t, model.DB.Create(&ch).Error)
	require.NoError(t, model.DB.Create(&model.Ability{Group: "default", Model: "gpt-5.6-sol", ChannelId: ch.Id, Enabled: true}).Error)
	engine := gin.New()
	engine.Use(middleware.RequestId())
	SetRelayRouter(engine)
	gateway := httptest.NewServer(engine)
	defer gateway.Close()
	cfg, work := t.TempDir(), t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(cfg, "config.toml"), []byte(fmt.Sprintf(`model = "gpt-5.6-sol"
model_provider = "fixture"
approval_policy = "never"
sandbox_mode = "read-only"
web_search = "disabled"
[analytics]
enabled = false
[model_providers.fixture]
name = "Local fixture"
base_url = "%s/v1"
wire_api = "responses"
env_key = "NEWAPI_TEST_TOKEN"
supports_websockets = true
request_max_retries = 0
stream_max_retries = 0
`, gateway.URL)), 0600))
	run := func(args ...string) string {
		t.Helper()
		ctx, cancel := context.WithTimeout(context.Background(), 45*time.Second)
		defer cancel()
		cmd := exec.CommandContext(ctx, bin, args...)
		cmd.Dir = work
		cmd.Env = []string{"PATH=" + os.Getenv("PATH"), "HOME=" + cfg, "CODEX_HOME=" + cfg, "TMPDIR=" + os.TempDir(), "LANG=en_US.UTF-8", "NEWAPI_TEST_TOKEN=sk-" + token.Key, "HTTP_PROXY=http://127.0.0.1:1", "HTTPS_PROXY=http://127.0.0.1:1", "NO_PROXY=127.0.0.1,localhost"}
		out, err := cmd.CombinedOutput()
		require.NoError(t, err, string(out))
		return string(out)
	}
	first := run("exec", "--skip-git-repo-check", "--json", "Use the available shell tool once and report its result.")
	require.Contains(t, first, "TOOL_RESULT_CONFIRMED", first)
	var session string
	for _, line := range strings.Split(first, "\n") {
		j := gjson.Parse(line)
		if j.Get("type").Str == "thread.started" {
			session = j.Get("thread_id").Str
		}
	}
	require.NotEmpty(t, session, first)
	later := run("exec", "resume", "--skip-git-repo-check", "--json", session, "RESUME_LOCAL_SESSION: confirm the preceding tool result.")
	require.Contains(t, later, "RESUME_CONFIRMED", later)
	ch.SetSetting(dto.ChannelSettings{})
	require.NoError(t, model.DB.Model(&ch).Update("setting", ch.Setting).Error)
	httpResume := run("exec", "resume", "--skip-git-repo-check", "--json", session, "RESUME_HTTP_SESSION: confirm the earlier tool result over the fallback route.")
	require.Contains(t, httpResume, "HTTP_RESUME_CONFIRMED", httpResume)
	mu.Lock()
	defer mu.Unlock()
	require.True(t, resumed)
	require.GreaterOrEqual(t, connections, 2)
	require.Equal(t, 1, toolResponses)
	require.Equal(t, 1, httpResumes)
	t.Logf("Codex resumed the same session after tool completion: connections=%d creates=%d http_resumes=%d", connections, calls, httpResumes)
}
func cliFixtureText(id, text string) map[string]any {
	return map[string]any{"type": "message", "id": id, "role": "assistant", "status": "completed", "content": []any{map[string]any{"type": "output_text", "text": text, "annotations": []any{}}}}
}

func cliFixtureTool(tools []gjson.Result) map[string]any {
	for _, tool := range tools {
		functions, ns := []gjson.Result{tool}, ""
		if tool.Get("type").Str == "namespace" {
			functions = tool.Get("tools").Array()
			ns = tool.Get("name").Str
		}
		for _, f := range functions {
			name := f.Get("name").Str
			item := map[string]any{"id": "fc_local", "call_id": "call_local", "name": name, "status": "completed"}
			if ns != "" {
				item["namespace"] = ns
			}
			switch name {
			case "exec":
				if f.Get("type").Str != "custom" {
					continue
				}
				item["type"] = "custom_tool_call"
				item["input"] = `text("LOCAL_WS_" + "TOOL_27")`
			case "exec_command", "shell_command", "shell":
				args := map[string]any{"cmd": "printf LOCAL_WS_TOOL_27"}
				if name == "shell_command" {
					args = map[string]any{"command": "printf LOCAL_WS_TOOL_27"}
				}
				if name == "shell" {
					args = map[string]any{"command": []string{"/bin/sh", "-c", "printf LOCAL_WS_TOOL_27"}}
				}
				encoded, _ := common.Marshal(args)
				item["type"] = "function_call"
				item["arguments"] = string(encoded)
			default:
				continue
			}
			return item
		}
	}
	return nil
}
