package relay

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/dto"
	"github.com/QuantumNous/new-api/pkg/websearch"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	relayconstant "github.com/QuantumNous/new-api/relay/constant"
	"github.com/QuantumNous/new-api/types"
	"github.com/alicebob/miniredis/v2"
	"github.com/gin-gonic/gin"
	"github.com/go-redis/redis/v8"
	"github.com/stretchr/testify/require"
)

func TestEightModelConversationsLeaveSearchAvailable(t *testing.T) {
	t.Setenv("UNIFIED_WEB_SEARCH_ENABLED", "true")
	user, token := setupMidjourneyBillingTest(t, 1000000, 1000000)
	cache := miniredis.RunT(t)
	oldRedis, oldSecret := common.RDB, common.CPAAuditIdentitySecret
	common.RDB = redis.NewClient(&redis.Options{Addr: cache.Addr()})
	common.CPAAuditIdentitySecret = "test-signing-secret"
	t.Cleanup(func() { common.RDB.Close(); common.RDB = oldRedis; common.CPAAuditIdentitySecret = oldSecret })

	var searches atomic.Int32
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/tools/search" {
			http.NotFound(w, r)
			return
		}
		searches.Add(1)
		w.Header().Set("X-CPA-Web-Provider", "deepseek")
		io.WriteString(w, `{"results":[{"title":"fixture","url":"https://example.com","summary":"found"}]}`)
	}))
	defer upstream.Close()
	client := &websearch.Client{Key: "fixture", BaseURL: upstream.URL, HTTP: upstream.Client()}
	getClient := func() (*websearch.Client, error) { return client, nil }
	request := func(path, body string, index int) (*gin.Context, *relaycommon.RelayInfo) {
		c, _ := gin.CreateTestContext(httptest.NewRecorder())
		c.Request = httptest.NewRequest(http.MethodPost, path, strings.NewReader(body))
		c.Request.Header.Set("Content-Type", "application/json")
		t.Cleanup(func() { common.CleanupBodyStorage(c) })
		return c, &relaycommon.RelayInfo{UserId: user.Id, TokenId: token.Id, TokenKey: token.Key, RequestId: fmt.Sprintf("fixture-%d", index),
			OriginModelName: "gpt-6.1-sol", UsingGroup: "default", TokenGroup: "default", StartTime: time.Now(),
			RelayFormat: types.RelayFormatOpenAIResponses, RelayMode: relayconstant.RelayModeResponses, ChannelMeta: &relaycommon.ChannelMeta{},
			UserSetting: dto.UserSetting{BillingPreference: "wallet_only"}, PriceData: types.PriceData{GroupRatioInfo: types.GroupRatioInfo{GroupRatio: 1}}}
	}
	var conversations sync.WaitGroup
	finish := make(chan struct{})
	defer func() { close(finish); conversations.Wait() }()
	for i := range 8 {
		body := `{"model":"gpt-6.1-sol","input":"write code","tools":[{"type":"web_search"}]}`
		c, info := request("/v1/responses", body, i)
		started := make(chan struct{})
		result := make(chan *types.NewAPIError, 1)
		conversations.Go(func() {
			result <- managedWebHelper(c, info, func() *types.NewAPIError {
				close(started)
				<-finish
				return webError(errors.New("fixture finished"), 502)
			}, getClient)
		})
		select {
		case <-started:
		case err := <-result:
			t.Fatalf("model %d did not start: %v", i, err)
		case <-time.After(3 * time.Second):
			t.Fatal("model did not start")
		}
	}

	// Only the ninth request uses a tool. All eight model callbacks are still blocked.
	body := `{"model":"gpt-6.1-sol","input":"look it up","tools":[{"type":"web_search"}]}`
	c, info := request("/v1/responses", body, 9)
	walletBefore, tokenBefore := persistedMidjourneyQuota(t, user.Id, token.Id)
	rounds := 0
	err := managedWebHelper(c, info, func() *types.NewAPIError {
		rounds++
		answer := `{"id":"r2","status":"completed","output":[{"type":"message","role":"assistant","content":[{"type":"output_text","text":"found"}]}]}`
		if rounds == 1 {
			answer = `{"id":"r1","status":"completed","output":[{"type":"function_call","call_id":"c1","name":"newapi_web_search","arguments":"{\"query\":\"fixture\"}"}]}`
		}
		_, writeErr := io.WriteString(c.Writer, answer)
		require.NoError(t, writeErr)
		info.DeferTextBilling(&dto.Usage{PromptTokens: 10, CompletionTokens: 2, TotalTokens: 12}, 12)
		return nil
	}, getClient)
	require.Equal(t, 2, rounds)
	require.Nil(t, err)
	require.EqualValues(t, 1, searches.Load())
	require.Equal(t, "completed", info.ManagedWeb.Outcome)
	require.Equal(t, 1, info.ManagedWeb.SearchRequests)
	walletAfter, tokenAfter := persistedMidjourneyQuota(t, user.Id, token.Id)
	require.Equal(t, 62500+2*12, walletBefore-walletAfter)
	require.Equal(t, 62500+2*12, tokenBefore-tokenAfter)
}

type blockedSearchBackend struct {
	webTestBackend
	started chan struct{}
}

func (b *blockedSearchBackend) Search(ctx context.Context, _ websearch.SearchRequest) ([]websearch.Hit, error) {
	b.started <- struct{}{}
	<-ctx.Done()
	return nil, ctx.Err()
}

// Exhaust the real shared tool limit, then make a model request that needs search
// after a billable first round. It must settle only that round and never replay.
func TestToolQueueTimeoutAfterModelUsage(t *testing.T) {
	ctx, cancel := context.WithCancel(t.Context())
	var active sync.WaitGroup
	defer func() { cancel(); active.Wait() }()
	busy := &blockedSearchBackend{started: make(chan struct{}, 8)}
	for range 8 {
		active.Go(func() {
			s := websearch.Session{Backend: busy}
			_, _ = s.Search(ctx, websearch.SearchRequest{Query: "held"})
		})
	}
	for range 8 {
		select {
		case <-busy.started:
		case <-time.After(3 * time.Second):
			t.Fatal("could not occupy search capacity")
		}
	}

	user, token := setupMidjourneyBillingTest(t, 1000000, 1000000)
	body := `{"model":"gpt-6.1-sol","input":"search","tools":[{"type":"web_search"}]}`
	c, info := newMidjourneyBillingContext("/v1/responses", body, "", user, token, "gpt-6.1-sol", relayconstant.RelayModeResponses)
	info.RelayFormat = types.RelayFormatOpenAIResponses
	info.ChannelMeta = &relaycommon.ChannelMeta{}
	info.UserSetting = dto.UserSetting{BillingPreference: "wallet_only"}
	info.PriceData.GroupRatioInfo.GroupRatio = 1
	t.Cleanup(func() { common.CleanupBodyStorage(c) })
	plan, err := websearch.NewPlan([]byte(body), "responses")
	require.NoError(t, err)
	backend := &webTestBackend{}
	store := webTestStore{}
	session := &websearch.Session{Backend: backend, Store: store}
	rounds := 0
	apiErr := runManagedWeb(c, info, func() *types.NewAPIError {
		rounds++
		info.DeferTextBilling(&dto.Usage{PromptTokens: 10, CompletionTokens: 2, TotalTokens: 12}, 12)
		_, err := io.WriteString(c.Writer, `{"id":"r1","status":"completed","output":[{"type":"function_call","id":"f1","call_id":"c1","name":"newapi_web_search","arguments":"{\"query\":\"fixture\"}"}]}`)
		require.NoError(t, err)
		return nil
	}, []byte(body), plan, session)
	require.NotNil(t, apiErr)
	require.Equal(t, 503, apiErr.StatusCode)
	require.True(t, types.IsSkipRetryError(apiErr))
	require.False(t, types.IsChannelPenaltyAllowed(apiErr))
	require.Equal(t, 1, rounds)
	require.Zero(t, backend.calls)
	require.Zero(t, session.Meter)
	require.Empty(t, store, "never save a capacity rejection as a tool result")
	wallet, remaining := persistedMidjourneyQuota(t, user.Id, token.Id)
	require.Equal(t, 1000000-12, wallet)
	require.Equal(t, wallet, remaining)
	require.Equal(t, "failed", info.ManagedWeb.Outcome)
	require.Zero(t, info.ManagedWeb.SearchRequests)
	info.Billing.Refund(c)
	walletAfterRefund, _ := persistedMidjourneyQuota(t, user.Id, token.Id)
	require.Equal(t, wallet, walletAfterRefund)
}
