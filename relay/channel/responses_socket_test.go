package channel

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

	relaycommon "github.com/QuantumNous/new-api/relay/common"
	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	"github.com/stretchr/testify/require"
	"github.com/tidwall/gjson"
)

func TestResponsesSocketManagedRoundsKeepTheVerifiedParent(t *testing.T) {
	var dials, frames atomic.Int32
	var upgrade websocket.Upgrader
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conn, err := upgrade.Upgrade(w, r, nil)
		if err != nil {
			return
		}
		defer conn.Close()
		dials.Add(1)
		for {
			_, b, err := conn.ReadMessage()
			if err != nil {
				return
			}
			n := frames.Add(1)
			if n == 2 {
				j := gjson.ParseBytes(b)
				results := j.Get("input").Array()
				if j.Get("previous_response_id").Str != "resp_internal_1" || len(results) != 1 || results[0].Get("call_id").Str != "lookup_1" {
					t.Error("managed result lost its parent or call ID")
					return
				}
			}
			id := fmt.Sprintf("resp_internal_%d", n)
			if err := conn.WriteMessage(websocket.TextMessage, []byte(fmt.Sprintf(`{"type":"response.completed","response":{"id":%q,"status":"completed","output":[]}}`, id))); err != nil {
				return
			}
		}
	}))
	defer upstream.Close()
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	socket := &ResponsesSocket{Context: ctx}
	defer socket.Close()
	turn := &ResponsesSocketTurn{Socket: socket}
	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	c.Request = httptest.NewRequestWithContext(ctx, "POST", "/v1/responses", nil)
	info := &relaycommon.RelayInfo{ChannelMeta: &relaycommon.ChannelMeta{ChannelId: 9, UpstreamModelName: "fixture"}, ManagedWeb: &relaycommon.ManagedWebBilling{}}
	a := &contextTestAdaptor{requestURL: upstream.URL + "/v1/responses"}
	send := func(raw string) {
		t.Helper()
		resp, err := turn.Request(c, a, info, strings.NewReader(raw))
		require.NoError(t, err)
		_, err = io.ReadAll(resp.Body)
		require.NoError(t, err)
		require.NoError(t, resp.Body.Close())
	}
	send(`{"model":"fixture","input":"first"}`)
	require.Error(t, turn.ContinueManagedRound("resp_foreign"))
	_, err := turn.Request(c, a, info, strings.NewReader(`{"model":"fixture","input":"replay"}`))
	require.Error(t, err)
	require.NoError(t, turn.ContinueManagedRound("resp_internal_1"))
	send(`{"model":"fixture","previous_response_id":"resp_internal_1","input":[{"type":"function_call_output","call_id":"lookup_1","output":"result"}]}`)
	require.EqualValues(t, 1, dials.Load())
	require.EqualValues(t, 2, frames.Load())
}
