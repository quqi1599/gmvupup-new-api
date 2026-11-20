package websearch

import (
	"context"
	"errors"
	"time"

	"golang.org/x/sync/semaphore"
)

// Search and fetch share the same provider account across all sessions in this process.
// Only actual tool calls wait here. A full queue or expired wait has no billable attempt.
var (
	activeToolCalls  = semaphore.NewWeighted(8)
	waitingToolCalls = make(chan struct{}, 8)
	ErrToolCapacity  = errors.New("managed web search capacity is temporarily exhausted")
)

func acquireToolSlot(ctx context.Context) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if activeToolCalls.TryAcquire(1) {
		return nil
	}
	select {
	case waitingToolCalls <- struct{}{}:
		defer func() { <-waitingToolCalls }()
	default:
		return ErrToolCapacity
	}
	waitCtx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()
	if err := activeToolCalls.Acquire(waitCtx, 1); err != nil {
		if ctx.Err() != nil {
			return ctx.Err()
		}
		return ErrToolCapacity
	}
	return nil
}
