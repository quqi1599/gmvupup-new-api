package websearch

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

type heldSearch struct {
	fakeBackend
	entered chan struct{}
	finish  <-chan struct{}
}

func (b *heldSearch) Search(ctx context.Context, _ SearchRequest) ([]Hit, error) {
	b.entered <- struct{}{}
	select {
	case <-b.finish:
		return []Hit{{Title: "fixture"}}, nil
	case <-ctx.Done():
		return nil, ctx.Err()
	}
}

// One session per caller, just as in the relay. Fetch must wait behind searches,
// then proceed immediately when a backend call finishes even if its session lives on.
func TestToolSlotsSharedAcrossSessionsAndOperations(t *testing.T) {
	ctx, cancel := context.WithCancel(t.Context())
	terminate := make(chan struct{})
	backend := &heldSearch{entered: make(chan struct{}, 8), finish: terminate}
	var calls sync.WaitGroup
	defer func() { cancel(); calls.Wait() }()
	for range 8 {
		calls.Go(func() { s := Session{Backend: backend}; _, _ = s.Search(ctx, SearchRequest{Query: "fixture"}) })
	}
	for range 8 {
		select {
		case <-backend.entered:
		case <-time.After(3 * time.Second):
			t.Fatal("search did not start")
		}
	}
	fetch := &Session{Backend: &fakeBackend{}}
	result := make(chan error, 1)
	calls.Go(func() { _, err := fetch.Fetch(ctx, "https://example.com"); result <- err })
	require.Eventually(t, func() bool { return len(waitingToolCalls) == 1 }, time.Second, time.Millisecond)
	terminate <- struct{}{}
	select {
	case err := <-result:
		require.NoError(t, err)
	case <-time.After(time.Second):
		t.Fatal("queued fetch did not use the released slot")
	}
	require.Equal(t, Meter{Fetch: 1, Attempts: 1}, fetch.Meter)
}

func TestToolQueueFullTimeoutAndCancellationDoNotDispatchOrReserve(t *testing.T) {
	require.True(t, activeToolCalls.TryAcquire(8))
	defer activeToolCalls.Release(8)
	ctx, cancel := context.WithCancel(t.Context())
	var calls sync.WaitGroup
	defer func() { cancel(); calls.Wait() }()
	// Every waiting session has an independent meter. A nil backend would panic if called.
	wait := func(ctx context.Context) error {
		s := Session{BeforeCall: func() error { t.Error("reserved quota for a rejected call"); return nil }}
		_, err := s.Search(ctx, SearchRequest{Query: "fixture"})
		if s.Meter != (Meter{}) {
			t.Errorf("meter changed before dispatch: %+v", s.Meter)
		}
		return err
	}
	results := make(chan error, 8)
	for range 8 {
		calls.Go(func() { results <- wait(ctx) })
	}
	require.Eventually(t, func() bool { return len(waitingToolCalls) == 8 }, time.Second, time.Millisecond)
	require.ErrorIs(t, wait(ctx), ErrToolCapacity)
	cancel()
	for range 8 {
		require.ErrorIs(t, <-results, context.Canceled)
	}
	require.Empty(t, waitingToolCalls)
	require.ErrorIs(t, wait(t.Context()), ErrToolCapacity)
	require.Empty(t, waitingToolCalls)
}

func TestToolPermitReleasedOnReservationFailure(t *testing.T) {
	sentinel := context.Canceled
	s := Session{BeforeCall: func() error { return sentinel }}
	_, err := s.Fetch(t.Context(), "https://example.com")
	require.ErrorIs(t, err, sentinel)
	require.Zero(t, s.Meter)
	require.True(t, activeToolCalls.TryAcquire(8))
	activeToolCalls.Release(8)
}
