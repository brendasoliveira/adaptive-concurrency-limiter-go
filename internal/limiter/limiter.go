package limiter

import (
	"context"
	"errors"
)

var ErrInvalidMaxConcurrency = errors.New("max concurrency must be greater than zero")

type ConcurrencyLimiter interface {
	AcquireSlot(ctx context.Context) bool
	ReleaseSlot()
}

type staticConcurrencyLimiter struct {
	activeSlots chan struct{}
}

func NewStaticConcurrencyLimiter(maxConcurrency int) (ConcurrencyLimiter, error) {
	if maxConcurrency <= 0 {
		return nil, ErrInvalidMaxConcurrency
	}

	return &staticConcurrencyLimiter{
		activeSlots: make(chan struct{}, maxConcurrency),
	}, nil
}

func (l *staticConcurrencyLimiter) AcquireSlot(ctx context.Context) bool {
	if ctx.Err() != nil {
		return false
	}

	select {
	case l.activeSlots <- struct{}{}:
		if ctx.Err() != nil {
			<-l.activeSlots
			return false
		}
		return true
	case <-ctx.Done():
		return false
	}
}

func (l *staticConcurrencyLimiter) ReleaseSlot() {
	select {
	case <-l.activeSlots:
	default:
		panic("ReleaseSlot called without an acquired slot")
	}
}
