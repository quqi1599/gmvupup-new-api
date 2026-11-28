package channel

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"slices"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/pkg/websearch"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	"github.com/QuantumNous/new-api/service"
	"github.com/QuantumNous/new-api/types"
	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	"github.com/tidwall/gjson"
)

type responseTurnKey struct{}

// The client connection owns the upstream socket. Only its one active turn can
// select a channel or read events. Control frames share the outbound writer.
type ResponsesSocket struct {
	Context       context.Context
	Binding       model.ResponsesSocketBinding
	mu            sync.Mutex
	conn          *websocket.Conn
	configuration [32]byte
	recent        [64]string // recently retired upstream IDs, including turns sent without a parent
	recentNext    int
}

type ResponsesSocketTurn struct {
	Socket              *ResponsesSocket
	Generate            *bool
	StreamID            string
	EventID             string
	Previous            string
	sent                bool // guarded by Socket.mu, together with currentID, terminal and pendingCancel
	currentID, terminal string
	pendingCancel       []byte
	broken              atomic.Bool
	// RelayOutcome is set by the single request worker after settlement.
	RelayOutcome       string
	cancelEvent        []byte
	upstreamStatus     int
	cancelID           string
	ReportControlError func(eventID string, status int) error
}

func (t *ResponsesSocketTurn) Context(ctx context.Context) context.Context {
	return context.WithValue(ctx, responseTurnKey{}, t)
}
func SocketTurn(ctx context.Context) *ResponsesSocketTurn {
	t, _ := ctx.Value(responseTurnKey{}).(*ResponsesSocketTurn)
	return t
}

func (s *ResponsesSocket) Close() {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.conn != nil {
		_ = s.conn.Close()
	}
}

// Cancel received during authentication must follow this turn's create, even
// when a preceding turn already opened the upstream connection.
func (t *ResponsesSocketTurn) Cancel(eventID, responseID string) error {
	s := t.Socket
	s.mu.Lock()
	defer s.mu.Unlock()
	if t.terminal != "" || t.broken.Load() {
		return errors.New("response is no longer active")
	}
	if t.cancelID != "" {
		return errors.New("a cancellation is already pending")
	}
	if responseID != "" && responseID != t.currentID {
		return errors.New("response_id does not match the active response")
	}
	// Construct the control from the parsed fields; only create frames enter
	// the model/billing path and no raw client frame bypasses that boundary.
	control := map[string]string{"type": "response.cancel"}
	if eventID == "" {
		eventID = common.NewRequestId()
	}
	if eventID == t.EventID {
		return errors.New("cancel event ID duplicates the create event ID")
	}
	control["event_id"] = eventID
	if responseID != "" {
		control["response_id"] = responseID
	}
	if t.StreamID != "" {
		control["stream_id"] = t.StreamID
	}
	data, err := common.Marshal(control)
	if err != nil {
		return err
	}
	t.cancelID = eventID
	if t.currentID == "" {
		t.pendingCancel = data
		return nil
	}
	_ = s.conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
	return s.conn.WriteMessage(websocket.TextMessage, data)
}

func (t *ResponsesSocketTurn) ConnectionLost() bool { return t.broken.Load() }

// FailureMetadata is read only after the stream handler has finished.
func (t *ResponsesSocketTurn) FailureMetadata() ([]byte, int) {
	t.Socket.mu.Lock()
	defer t.Socket.mu.Unlock()
	return t.cancelEvent, t.upstreamStatus
}

// Only the gateway's managed tool loop may authorize another generation
// within one billable client request, after its preceding model round finished.
func (t *ResponsesSocketTurn) ContinueManagedRound(parent string) error {
	s := t.Socket
	s.mu.Lock()
	defer s.mu.Unlock()
	if !t.sent || parent == "" || parent != t.currentID || (t.terminal != "response.completed" && t.terminal != "response.done") || t.broken.Load() {
		return errors.New("managed tool result has no completed upstream parent")
	}
	t.Previous = parent
	t.sent = false
	t.currentID = ""
	t.terminal = ""
	t.pendingCancel = nil
	t.cancelID = ""
	return nil
}

// Request adapts native JSON frames to the existing Responses stream reader.
// Usage, tool fees, incomplete responses and refunds use the HTTP relay path.
func (t *ResponsesSocketTurn) Request(c *gin.Context, a Adaptor, info *relaycommon.RelayInfo, body io.Reader) (*http.Response, error) {
	s := t.Socket
	raw, err := io.ReadAll(io.LimitReader(body, int64(maxSocketBody())+1))
	if err != nil {
		return nil, err
	}
	if len(raw) > maxSocketBody() {
		return nil, errors.New("Responses WebSocket request is too large")
	}
	if info.ManagedWeb != nil {
		plan, err := websearch.NewPlan(raw, "responses")
		if err != nil {
			return nil, err
		}
		if plan != nil {
			return nil, errors.New("model channel attempted to re-enable a vendor hosted web tool")
		}
	}
	var request map[string]json.RawMessage
	if err := common.Unmarshal(raw, &request); err != nil || request == nil {
		return nil, errors.New("invalid adapted WebSocket request")
	}
	if gjson.GetBytes(request["previous_response_id"], "@this").String() != t.Previous {
		return nil, errors.New("channel rewrote the WebSocket parent response")
	}
	for _, field := range []string{"stream", "stream_options", "background", "event_id", "generate", "stream_id"} {
		delete(request, field)
	}
	request["type"] = json.RawMessage(`"response.create"`)
	if t.Generate != nil {
		request["generate"], _ = common.Marshal(*t.Generate)
	}
	if t.StreamID != "" {
		request["stream_id"], _ = common.Marshal(t.StreamID)
	}
	if t.EventID != "" {
		request["event_id"], _ = common.Marshal(t.EventID)
	}
	raw, err = common.Marshal(request)
	if err != nil {
		return nil, err
	}
	full, err := getRequestURL(a, c, info)
	if err != nil {
		return nil, err
	}
	headers, err := responsesHandshakeHeaders(c, a, info)
	if err != nil {
		return nil, err
	}
	// Affinity and tracing may add per-turn client headers after warmup. Only
	// changes to the physical endpoint, channel configuration or resolved
	// credentials change the owner of an existing connection.
	authority := map[string]string{}
	for _, name := range []string{"Host", "Authorization", "Proxy-Authorization", "Api-Key", "X-Api-Key", "ChatGPT-Account-Id", "OpenAI-Organization", "OpenAI-Project", "Cookie"} {
		if v := headers.Get(name); v != "" {
			authority[name] = v
		}
	}
	identity, err := common.Marshal([]any{full, info.ChannelType, info.UpstreamModelName, info.Organization, info.HeadersOverride, authority, info.ChannelSetting.Proxy, info.ChannelOtherSettings.CPAAuditIdentityEnabled})
	if err != nil {
		return nil, err
	}
	fingerprint := sha256.Sum256(identity)

	s.mu.Lock()
	if t.sent {
		s.mu.Unlock()
		return nil, errors.New("Responses WebSocket request cannot be replayed")
	}
	if s.conn == nil {
		conn, err := dialResponsesSocket(s.Context, c, info, full, headers)
		if err != nil {
			s.mu.Unlock()
			return nil, err
		}
		s.conn = conn
		s.configuration = fingerprint
		group := info.UsingGroup
		if g := common.GetContextKeyString(c, constant.ContextKeyAutoGroup); g != "" {
			group = g
		}
		s.Binding = model.ResponsesSocketBinding{ChannelID: info.ChannelId, Model: info.OriginModelName, Group: group, KeyIndex: info.ChannelMultiKeyIndex, KeyHash: sha256.Sum256([]byte(info.ApiKey)), TokenID: info.TokenId, UserID: info.UserId}
	} else if s.configuration != fingerprint {
		s.mu.Unlock()
		return nil, types.NewErrorWithStatusCode(errors.New("Responses WebSocket route configuration changed; reconnect with complete history"), types.ErrorCodeGetChannelFailed, http.StatusConflict, types.ErrOptionWithSkipRetry())
	}
	conn := s.conn
	common.SetContextKey(c, constant.ContextKeyRelayConnectedUpstream, true)
	common.SetContextKey(c, constant.ContextKeyRelayResponseHeaders, true)
	t.sent = true
	info.MarkUpstreamRequestMayHaveBeenAccepted()
	_ = conn.SetWriteDeadline(time.Now().Add(30 * time.Second))
	err = conn.WriteMessage(websocket.TextMessage, raw)
	s.mu.Unlock()
	if err != nil {
		t.broken.Store(true)
		s.Close()
		return nil, errors.New("upstream WebSocket write failed")
	}
	common.SetContextKey(c, constant.ContextKeyRelayRequestWritten, true)
	common.ReleaseBodyAdmission(c)
	events := &socketEvents{turn: t, conn: conn, streamID: t.StreamID, previous: t.Previous, warmup: t.Generate != nil && !*t.Generate}
	return &http.Response{StatusCode: http.StatusOK, Header: http.Header{"Content-Type": []string{"text/event-stream"}}, Body: events}, nil
}

func maxSocketBody() int {
	if constant.MaxRequestBodyMB <= 0 {
		return 128 << 20
	}
	return constant.MaxRequestBodyMB << 20
}

func responsesHandshakeHeaders(c *gin.Context, a Adaptor, info *relaycommon.RelayInfo) (http.Header, error) {
	h := make(http.Header)
	if err := a.SetupRequestHeader(c, &h, info); err != nil {
		return nil, err
	}
	override, err := processHeaderOverride(info, c)
	if err != nil {
		return nil, err
	}
	for k, v := range override {
		h.Set(k, v)
	}
	for k := range h {
		if strings.HasPrefix(strings.ToLower(k), "sec-websocket-") {
			h.Del(k)
		}
	}
	for _, name := range []string{"Connection", "Upgrade", "Content-Length", "Accept-Encoding", "Content-Encoding"} {
		h.Del(name)
	}
	h.Set("OpenAI-Beta", "responses_websockets=2026-02-06")
	applyCPAPrincipal(h, info)
	if err := requirePlaygroundCPAPrincipal(h, info); err != nil {
		return nil, err
	}
	return h, nil
}

func dialResponsesSocket(lifetime context.Context, c *gin.Context, info *relaycommon.RelayInfo, full string, h http.Header) (*websocket.Conn, error) {
	u, err := url.Parse(full)
	if err != nil {
		return nil, err
	}
	switch u.Scheme {
	case "http":
		u.Scheme = "ws"
	case "https":
		u.Scheme = "wss"
	case "ws", "wss":
	default:
		return nil, errors.New("invalid Responses WebSocket upstream scheme")
	}
	if err := applyCPAAuditIdentityHeaders(h, http.MethodGet, full, info, time.Now(), common.CPAAuditIdentitySecret); err != nil {
		return nil, err
	}
	dialer := newRelayWebSocketDialer()
	if rawProxy := info.ChannelSetting.Proxy; rawProxy != "" {
		client, err := service.GetHttpClientWithProxy(rawProxy)
		if err != nil {
			return nil, err
		}
		tr, ok := client.Transport.(*http.Transport)
		if !ok {
			return nil, errors.New("unsupported WebSocket proxy transport")
		}
		dialer.Proxy = tr.Proxy
		dialer.NetDialContext = tr.DialContext
	}
	configureDirectRelayWebSocketTLS(c.Request.Context(), u.String(), &dialer)
	// Link even a pending handshake to the client's socket lifetime.
	ctx, cancel := context.WithCancel(c.Request.Context())
	stop := context.AfterFunc(lifetime, cancel)
	defer stop()
	defer cancel()
	conn, resp, err := dialer.DialContext(ctx, u.String(), h)
	if err != nil {
		if resp != nil {
			service.CloseResponseBodyGracefully(resp)
			if resp.StatusCode >= 400 && resp.StatusCode <= 599 {
				return nil, types.NewErrorWithStatusCode(fmt.Errorf("upstream Responses WebSocket handshake returned HTTP %d", resp.StatusCode), types.ErrorCodeDoRequestFailed, resp.StatusCode, types.ErrOptionWithUpstreamResponse())
			}
			return nil, types.NewErrorWithStatusCode(errors.New("upstream did not upgrade to Responses WebSocket"), types.ErrorCodeDoRequestFailed, http.StatusBadGateway)
		}
		if ctx.Err() != nil {
			return nil, ClassifyDoRequestError(c, ctx.Err())
		}
		return nil, types.NewErrorWithStatusCode(errors.New("upstream Responses WebSocket handshake failed"), types.ErrorCodeDoRequestFailed, http.StatusBadGateway)
	}
	conn.SetReadLimit(int64(maxSocketBody()))
	return conn, nil
}

type socketEvents struct {
	seenCreated                    bool
	turn                           *ResponsesSocketTurn
	conn                           *websocket.Conn
	streamID, previous, responseID string
	warmup                         bool
	buffer                         bytes.Reader
	terminal                       atomic.Bool
}

func (r *socketEvents) Close() error {
	if !r.terminal.Load() {
		r.turn.broken.Store(true)
		return r.conn.Close()
	}
	return nil
}
func (r *socketEvents) Read(p []byte) (int, error) {
	if r.buffer.Len() > 0 {
		return r.buffer.Read(p)
	}
	if r.terminal.Load() {
		return 0, io.EOF
	}
	for {
		kind, data, err := r.conn.ReadMessage()
		if err != nil {
			return 0, errors.New("upstream Responses WebSocket closed before a terminal event")
		}
		if kind != websocket.TextMessage || !gjson.ValidBytes(data) {
			return 0, errors.New("invalid upstream Responses WebSocket event")
		}
		obj := gjson.ParseBytes(data)
		event := obj.Get("type").Str
		id := obj.Get("response.id").Str
		if id == "" {
			id = obj.Get("response_id").Str
		}
		if event == "" {
			return 0, errors.New("upstream Responses WebSocket event has no type")
		}
		if lane := obj.Get("stream_id").Str; lane != "" && lane != r.streamID {
			return 0, errors.New("upstream Responses WebSocket stream_id mismatch")
		}
		if id != "" && event != "error" {
			socket := r.turn.Socket
			socket.mu.Lock()
			retired := slices.Contains(socket.recent[:], id)
			socket.mu.Unlock()
			if retired || id == r.previous {
				if event == "response.created" {
					return 0, errors.New("upstream reused a retired response ID")
				}
				continue
			}
		}
		if event == "error" && r.turn.ReportControlError != nil {
			ref := obj.Get("error.event_id").Str
			if ref == "" {
				ref = obj.Get("event_id").Str
			}
			r.turn.Socket.mu.Lock()
			matched := ref != "" && ref == r.turn.cancelID
			if matched {
				r.turn.cancelID = ""
			}
			r.turn.Socket.mu.Unlock()
			if matched {
				status := int(obj.Get("status").Int())
				if status < 400 || status > 599 {
					status = 400
				}
				if err := r.turn.ReportControlError(ref, status); err != nil {
					return 0, err
				}
				continue
			}
		}
		if id != "" && r.responseID != "" && id != r.responseID {
			return 0, errors.New("upstream Responses WebSocket response ID changed")
		}
		if event == "response.created" {
			if id == "" {
				return 0, errors.New("upstream created event has no response ID")
			}
			if r.seenCreated {
				continue
			}
			r.seenCreated = true
		}
		if id != "" && r.responseID == "" {
			r.responseID = id
			r.turn.Socket.mu.Lock()
			r.turn.currentID = id
			if pending := r.turn.pendingCancel; pending != nil && (event == "response.created" || event == "response.in_progress") {
				_ = r.conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
				err = r.conn.WriteMessage(websocket.TextMessage, pending)
				r.turn.pendingCancel = nil
			}
			r.turn.Socket.mu.Unlock()
			if err != nil {
				return 0, err
			}
		}
		if r.warmup && (strings.HasPrefix(event, "response.output_") || strings.HasPrefix(event, "response.function_call") || strings.HasPrefix(event, "response.content_part")) {
			return 0, errors.New("upstream generated output during WebSocket warmup")
		}
		switch event {
		case "response.completed", "response.done", "response.incomplete":
			if responseID := obj.Get("response.id"); responseID.Type != gjson.String || responseID.Str == "" || len(responseID.Str) > 1024 {
				return 0, errors.New("upstream terminal event has no valid response ID")
			}
			if r.warmup && (len(obj.Get("response.output").Array()) != 0 || obj.Get("response.usage.output_tokens").Int() > 0 || event == "response.incomplete") {
				return 0, errors.New("upstream failed WebSocket non-generating warmup")
			}
			r.terminal.Store(true)
		case "error", "response.error", "response.failed", "response.cancelled", "response.canceled":
			r.terminal.Store(true)
		}
		if r.terminal.Load() {
			r.turn.Socket.mu.Lock()
			r.turn.terminal = event
			if id != "" {
				s := r.turn.Socket
				s.recent[s.recentNext] = id
				s.recentNext = (s.recentNext + 1) % len(s.recent)
			}
			if event == "response.cancelled" || event == "response.canceled" {
				r.turn.cancelEvent = bytes.Clone(data)
			}
			if status := obj.Get("status").Int(); status >= 400 && status < 600 {
				r.turn.upstreamStatus = int(status)
			}
			r.turn.Socket.mu.Unlock()
		}
		// SSE uses a single data line here, even if the upstream pretty-prints JSON.
		if bytes.ContainsAny(data, "\r\n") {
			data, err = common.Marshal(json.RawMessage(data))
			if err != nil {
				return 0, err
			}
		}
		r.buffer.Reset(append(append([]byte("data: "), data...), '\n', '\n'))
		return r.buffer.Read(p)
	}
}
