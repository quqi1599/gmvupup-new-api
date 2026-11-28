package controller

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"
	"unicode/utf8"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	"github.com/QuantumNous/new-api/middleware"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/relay/channel"
	"github.com/QuantumNous/new-api/service"
	"github.com/QuantumNous/new-api/types"
	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	"github.com/tidwall/gjson"
	"github.com/tidwall/sjson"
)

var responsesUpgrader = websocket.Upgrader{Subprotocols: []string{"responses"}}
var responsesStreamID = regexp.MustCompile("^[A-Za-z0-9_.-]{1,256}$")

// As in QuantumNous/new-api #5062, the handshake authenticates a connection.
// Each response.create runs fresh request middleware and settles independently.
var responsesTurnEngine = sync.OnceValue(func() *gin.Engine {
	e := gin.New()
	e.ForwardedByClientIP = false
	_ = e.SetTrustedProxies(nil)
	e.POST("/v1/responses", middleware.RequestId(), middleware.BodyStorageCleanup(), middleware.SystemPerformanceCheck(), middleware.TokenAuth(),
		func(c *gin.Context) {
			t := channel.SocketTurn(c.Request.Context())
			common.SetContextKey(c, constant.ContextKeyResponsesWebSocket, &t.Socket.Binding)
			common.SetContextKey(c, constant.ContextKeyRequestStartTime, time.Now())
			c.Next()
		},
		middleware.ModelPerformance(), middleware.ModelRequestRateLimit(), middleware.Distribute(),
		func(c *gin.Context) {
			Relay(c, types.RelayFormatOpenAIResponses)
			channel.SocketTurn(c.Request.Context()).RelayOutcome = c.GetString("relay_final_outcome")
		})
	return e
})

type socketEnvelope struct {
	Type       string          `json:"type"`
	EventID    string          `json:"event_id"`
	StreamID   string          `json:"stream_id"`
	Response   json.RawMessage `json:"response"`
	ResponseID string          `json:"response_id"`
	Generate   *bool           `json:"generate"`
}

func (e *socketEnvelope) request(raw []byte) ([]byte, error) {
	if len(e.Response) > 0 {
		raw = e.Response
	}
	var body map[string]json.RawMessage
	if err := common.Unmarshal(raw, &body); err != nil || body == nil {
		return nil, errors.New("response.create requires a request object")
	}
	if len(e.Response) > 0 {
		var nested socketEnvelope
		if err := common.Unmarshal(raw, &nested); err != nil {
			return nil, err
		}
		if e.Generate == nil {
			e.Generate = nested.Generate
		}
		if e.StreamID == "" {
			e.StreamID = nested.StreamID
		} else if nested.StreamID != "" && nested.StreamID != e.StreamID {
			return nil, errors.New("conflicting stream_id")
		}
	}
	if e.StreamID != "" && !responsesStreamID.MatchString(e.StreamID) {
		return nil, errors.New("invalid stream_id")
	}
	for _, k := range []string{"type", "generate", "stream_id", "event_id", "stream_options", "background"} {
		delete(body, k)
	}
	body["stream"] = json.RawMessage("true")
	if _, found := body["input"]; !found && e.Generate != nil && !*e.Generate {
		body["input"] = json.RawMessage("[]")
	}
	return common.Marshal(body)
}

type responsesClient struct {
	ws      *websocket.Conn
	writeMu sync.Mutex
}

func (s *responsesClient) write(data []byte) error {
	s.writeMu.Lock()
	defer s.writeMu.Unlock()
	_ = s.ws.SetWriteDeadline(time.Now().Add(30 * time.Second))
	return s.ws.WriteMessage(websocket.TextMessage, data)
}
func (s *responsesClient) fail(e socketEnvelope, status int, detail any) error {
	msg := map[string]any{"type": "error", "status": status, "error": detail}
	if e.EventID != "" {
		msg["event_id"] = e.EventID
	}
	if e.StreamID != "" {
		msg["stream_id"] = e.StreamID
	}
	raw, err := common.Marshal(msg)
	if err != nil {
		return err
	}
	return s.write(raw)
}
func wsDetail(code, message string) map[string]string {
	return map[string]string{"type": "invalid_request_error", "code": code, "message": message}
}

func ResponsesWebSocket(c *gin.Context) {
	if !websocket.IsWebSocketUpgrade(c.Request) {
		c.JSON(426, gin.H{"error": wsDetail("upgrade_required", "Responses WebSocket requires an upgrade")})
		return
	}
	pin, _ := strconv.Atoi(c.GetString(string(constant.ContextKeyTokenSpecificChannelId)))
	var allowedModels []string
	if common.GetContextKeyBool(c, constant.ContextKeyTokenModelLimitEnabled) {
		allowedModels = []string{}
		if v, ok := c.Get(string(constant.ContextKeyTokenModelLimit)); ok {
			if m, ok := v.(map[string]bool); ok {
				for k := range m {
					allowedModels = append(allowedModels, k)
				}
			}
		}
	}
	ok, err := model.HasResponsesSocketChannel(service.ResponsesSocketGroups(c), pin, allowedModels)
	if err != nil {
		c.JSON(503, gin.H{"error": wsDetail("channel_lookup_failed", "Responses WebSocket routes are unavailable")})
		return
	}
	if !ok {
		c.JSON(426, gin.H{"error": wsDetail("websocket_not_supported", "Use HTTP Responses for this provider")})
		return
	}
	ws, err := responsesUpgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		return
	}
	maxBytes := constant.MaxRequestBodyMB
	if maxBytes <= 0 {
		maxBytes = 128
	}
	ws.SetReadLimit(int64(maxBytes) << 20)
	ctx, cancel := context.WithTimeout(c.Request.Context(), time.Hour)
	transport := &channel.ResponsesSocket{Context: ctx}
	client := &responsesClient{ws: ws}
	stop := context.AfterFunc(ctx, func() { _ = ws.Close(); transport.Close() })
	var workers sync.WaitGroup
	defer func() { cancel(); _ = ws.Close(); transport.Close(); workers.Wait(); stop() }()

	headers := c.Request.Header.Clone()
	for name := range headers {
		if strings.HasPrefix(strings.ToLower(name), "sec-websocket-") {
			headers.Del(name)
		}
	}
	for _, name := range []string{"Connection", "Upgrade", "Content-Length", "Content-Encoding", common.RequestIdKey, "X-Request-Id", "X-Trace-Id", "X-B3-Traceid", common.AttemptIdKey} {
		headers.Del(name)
	}
	headers.Set("Content-Type", "application/json")
	peer := net.JoinHostPort(c.ClientIP(), "0")
	var mu sync.Mutex
	var active *channel.ResponsesSocketTurn
	parents := map[string]string{}
	for {
		kind, data, err := ws.ReadMessage()
		if err != nil {
			return
		}
		var e socketEnvelope
		if kind != websocket.TextMessage || !utf8.Valid(data) || common.Unmarshal(data, &e) != nil {
			if client.fail(e, 400, wsDetail("invalid_request_error", "expected a JSON text event")) != nil {
				return
			}
			continue
		}
		if e.Type == "response.cancel" {
			mu.Lock()
			turn := active
			mu.Unlock()
			if turn == nil || turn.StreamID != e.StreamID || turn.Cancel(e.EventID, e.ResponseID) != nil {
				if client.fail(e, 400, wsDetail("no_active_response", "no matching active response")) != nil {
					return
				}
			}
			continue
		}
		if e.Type != "response.create" {
			if client.fail(e, 400, wsDetail("invalid_request_error", "unsupported Responses WebSocket event")) != nil {
				return
			}
			continue
		}
		body, err := e.request(data)
		if err != nil {
			if client.fail(e, 400, wsDetail("invalid_request_error", err.Error())) != nil {
				return
			}
			continue
		}
		parent := gjson.GetBytes(body, "previous_response_id").String()
		mu.Lock()
		if active != nil {
			mu.Unlock()
			if client.fail(e, 409, wsDetail("response_in_progress", "a response is still in progress")) != nil {
				return
			}
			continue
		}
		if parent != "" && parents[e.StreamID] != parent {
			mu.Unlock()
			if client.fail(e, 400, wsDetail("previous_response_not_found", "Send complete input history to resume this response")) != nil {
				return
			}
			continue
		}
		if _, seen := parents[e.StreamID]; !seen && len(parents) >= 64 {
			mu.Unlock()
			if client.fail(e, 429, wsDetail("stream_limit", "too many streams on this connection")) != nil {
				return
			}
			continue
		}
		t := &channel.ResponsesSocketTurn{Socket: transport, Generate: e.Generate, StreamID: e.StreamID, EventID: e.EventID, Previous: parent}
		t.ReportControlError = func(eventID string, status int) error {
			control := socketEnvelope{EventID: eventID, StreamID: e.StreamID}
			return client.fail(control, status, wsDetail("cancel_rejected", "Upstream rejected the cancellation"))
		}
		active = t
		mu.Unlock()
		workers.Add(1)
		go func() {
			defer workers.Done()
			defer func() {
				if recover() != nil {
					common.SysError("panic in Responses WebSocket request")
					_ = client.fail(e, 500, wsDetail("internal_error", "Responses request interrupted"))
					cancel()
				}
			}()
			w := client.runTurn(ctx, t, e, body, headers, peer)
			mu.Lock()
			defer mu.Unlock()
			active = nil
			parentID, err := w.finish(t)
			if parentID != "" {
				parents[e.StreamID] = parentID
			}
			if err != nil || t.ConnectionLost() {
				cancel()
			}
		}()
	}
}

func (s *responsesClient) runTurn(ctx context.Context, t *channel.ResponsesSocketTurn, e socketEnvelope, body []byte, headers http.Header, peer string) *responsesSocketWriter {
	req, _ := http.NewRequestWithContext(t.Context(ctx), http.MethodPost, "/v1/responses", bytes.NewReader(body))
	req.Header = headers.Clone()
	req.RemoteAddr = peer
	w := &responsesSocketWriter{client: s, header: make(http.Header), envelope: e}
	responsesTurnEngine().ServeHTTP(w, req)
	return w
}

// Both publication and updating the lane's parent happen while the connection
// lock is held. A tool result can arrive as soon as this terminal is written.
func (w *responsesSocketWriter) finish(t *channel.ResponsesSocketTurn) (string, error) {
	cancelled, status := t.FailureMetadata()
	if cancelled != nil {
		w.terminal = cancelled
		w.kind = "response.cancelled"
	}
	var parentID string
	if w.kind == "response.completed" || w.kind == "response.done" || w.kind == "response.incomplete" {
		if t.RelayOutcome != "completed" && t.RelayOutcome != "incomplete" {
			w.terminal = nil
		} else {
			parentID = gjson.GetBytes(w.terminal, "response.id").Str
		}
	}
	if w.terminal != nil {
		payload, err := w.withStreamID(w.terminal)
		if err != nil {
			return "", err
		}
		return parentID, w.client.write(payload)
	}
	if status == 0 {
		status = w.status
	}
	if status < 400 {
		status = 502
	}
	var parsed struct {
		Error json.RawMessage `json:"error"`
	}
	var detail any = wsDetail("stream_interrupted", "Responses stream ended without a terminal event")
	if common.Unmarshal(w.failure.Bytes(), &parsed) == nil && len(parsed.Error) > 0 {
		detail = parsed.Error
	}
	return "", w.client.fail(w.envelope, status, detail)
}

func (w *responsesSocketWriter) withStreamID(data []byte) ([]byte, error) {
	if w.envelope.StreamID == "" || gjson.GetBytes(data, "stream_id").Exists() {
		return data, nil
	}
	return sjson.SetBytes(data, "stream_id", w.envelope.StreamID)
}

// The existing relay writes one SSE stream per request. Only JSON data frames
// cross the socket. Protocol terminal events wait for the relay to finish.
type responsesSocketWriter struct {
	client           *responsesClient
	envelope         socketEnvelope
	header           http.Header
	status           int
	pending, failure bytes.Buffer
	terminal         []byte
	kind             string
}

func (w *responsesSocketWriter) Header() http.Header { return w.header }
func (w *responsesSocketWriter) WriteHeader(code int) {
	if w.status == 0 {
		w.status = code
	}
}
func (w *responsesSocketWriter) Flush() {}
func (w *responsesSocketWriter) Write(p []byte) (int, error) {
	w.WriteHeader(http.StatusOK)
	if !strings.HasPrefix(w.header.Get("Content-Type"), "text/event-stream") {
		if w.failure.Len()+len(p) > 1<<20 {
			return 0, errors.New("oversized error response")
		}
		return w.failure.Write(p)
	}
	w.pending.Write(p)
	for {
		b := w.pending.Bytes()
		idx := bytes.Index(b, []byte("\n\n"))
		if idx < 0 {
			break
		}
		frame := w.pending.Next(idx + 2)
		for _, line := range bytes.Split(frame, []byte("\n")) {
			if !bytes.HasPrefix(line, []byte("data:")) {
				continue
			}
			data := bytes.TrimSpace(line[len("data:"):])
			if len(data) == 0 || bytes.Equal(data, []byte("[DONE]")) {
				continue
			}
			if !gjson.ValidBytes(data) {
				return 0, errors.New("invalid Responses data frame")
			}
			event := gjson.GetBytes(data, "type").Str
			switch event {
			case "response.completed", "response.done", "response.incomplete", "response.failed", "response.cancelled", "response.canceled", "error":
				w.terminal = bytes.Clone(data)
				w.kind = event
			default:
				payload, err := w.withStreamID(data)
				if err != nil {
					return 0, err
				}
				if err := w.client.write(payload); err != nil {
					return 0, err
				}
			}
		}
	}
	return len(p), nil
}
