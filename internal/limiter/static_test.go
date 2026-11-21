package limiter

import (
	"context"
	"strconv"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func TestStaticConcurrencyLimiterAcquireSlotAndReleaseSlot(t *testing.T) {
	limiter, err := NewStaticConcurrencyLimiter(1)
	if err != nil {
		t.Fatalf("NewStaticConcurrencyLimiter() error = %v", err)
	}

	if !limiter.AcquireSlot(t.Context()) {
		t.Fatal("AcquireSlot() = false, want true")
	}

	limiter.ReleaseSlot()

	if !limiter.AcquireSlot(t.Context()) {
		t.Fatal("AcquireSlot() after ReleaseSlot() = false, want true")
	}
	limiter.ReleaseSlot()
}

func TestStaticConcurrencyLimiterRejectsInvalidMaxConcurrency(t *testing.T) {
	for _, limit := range []int{0, -1} {
		t.Run("limit="+strconv.Itoa(limit), func(t *testing.T) {
			limiter, err := NewStaticConcurrencyLimiter(limit)
			if err == nil {
				t.Fatalf("NewStaticConcurrencyLimiter(%d) error = nil, want an invalid limit error", limit)
			}
			if limiter != nil {
				t.Fatalf("NewStaticConcurrencyLimiter(%d) limiter = %v, want nil", limit, limiter)
			}
		})
	}
}

func TestStaticConcurrencyLimiterAcquireSlotReturnsFalseWhenCanceled(t *testing.T) {
	limiter, err := NewStaticConcurrencyLimiter(1)
	if err != nil {
		t.Fatalf("NewStaticConcurrencyLimiter() error = %v", err)
	}

	if !limiter.AcquireSlot(t.Context()) {
		t.Fatal("initial AcquireSlot() = false, want true")
	}

	ctx, cancel := context.WithCancel(t.Context())
	acquisitionStarted := make(chan struct{})
	acquisitionResult := make(chan bool, 1)

	go func() {
		close(acquisitionStarted)
		acquisitionResult <- limiter.AcquireSlot(ctx)
	}()

	<-acquisitionStarted
	cancel()

	select {
	case acquired := <-acquisitionResult:
		if acquired {
			t.Fatal("AcquireSlot() = true after context cancellation, want false")
		}
	case <-time.After(time.Second):
		t.Fatal("AcquireSlot() did not return after context cancellation")
	}

	limiter.ReleaseSlot()
}

func TestStaticConcurrencyLimiterAcquireSlotReturnsFalseForAlreadyCanceledContext(t *testing.T) {
	limiter, err := NewStaticConcurrencyLimiter(1)
	if err != nil {
		t.Fatalf("NewStaticConcurrencyLimiter() error = %v", err)
	}

	ctx, cancel := context.WithCancel(t.Context())
	cancel()

	if limiter.AcquireSlot(ctx) {
		t.Fatal("AcquireSlot() = true for an already canceled context, want false")
	}

	if limiter.AcquireSlot(t.Context()) != true {
		t.Fatal("AcquireSlot() = false after rejected acquisition, want the slot to remain available")
	}
	limiter.ReleaseSlot()
}

func TestStaticConcurrencyLimiterCanceledAcquisitionDoesNotConsumeSlot(t *testing.T) {
	limiter, err := NewStaticConcurrencyLimiter(1)
	if err != nil {
		t.Fatalf("NewStaticConcurrencyLimiter() error = %v", err)
	}

	ctx, cancel := context.WithCancel(t.Context())
	cancel()

	if limiter.AcquireSlot(ctx) {
		t.Fatal("AcquireSlot() = true for canceled context, want false")
	}

	if !limiter.AcquireSlot(t.Context()) {
		t.Fatal("AcquireSlot() = false after canceled acquisition, want the slot to remain available")
	}

	limiter.ReleaseSlot()
}

func TestStaticConcurrencyLimiterReleaseSlotWithoutAcquisitionPanics(t *testing.T) {
	limiter, err := NewStaticConcurrencyLimiter(1)
	if err != nil {
		t.Fatalf("NewStaticConcurrencyLimiter() error = %v", err)
	}

	defer func() {
		if recovered := recover(); recovered == nil {
			t.Fatal("ReleaseSlot() did not panic without an acquired slot")
		}
	}()

	limiter.ReleaseSlot()
}

func TestStaticConcurrencyLimiterAcquireSlotReturnsFalseWhenDeadlineExpires(t *testing.T) {
	limiter, err := NewStaticConcurrencyLimiter(1)
	if err != nil {
		t.Fatalf("NewStaticConcurrencyLimiter() error = %v", err)
	}

	if !limiter.AcquireSlot(t.Context()) {
		t.Fatal("initial AcquireSlot() = false, want true")
	}

	ctx, cancel := context.WithTimeout(t.Context(), 20*time.Millisecond)
	defer cancel()

	if limiter.AcquireSlot(ctx) {
		t.Fatal("AcquireSlot() = true while the limit is occupied, want false")
	}

	if ctx.Err() != context.DeadlineExceeded {
		t.Fatalf("ctx.Err() = %v, want %v", ctx.Err(), context.DeadlineExceeded)
	}

	limiter.ReleaseSlot()
}

func TestStaticConcurrencyLimiterNeverAdmitsMoreThanMaxConcurrency(t *testing.T) {
	const (
		maxConcurrency = 3
		operationCount = 30
	)

	limiter, err := NewStaticConcurrencyLimiter(maxConcurrency)
	if err != nil {
		t.Fatalf("NewStaticConcurrencyLimiter() error = %v", err)
	}

	ctx, cancel := context.WithTimeout(t.Context(), time.Second)
	defer cancel()

	start := make(chan struct{})
	release := make(chan struct{})
	admitted := make(chan struct{}, operationCount)

	var active atomic.Int32
	var peak atomic.Int32
	var wg sync.WaitGroup

	for range operationCount {
		wg.Go(func() {
			<-start
			if !limiter.AcquireSlot(ctx) {
				return
			}

			current := active.Add(1)
			for {
				previousPeak := peak.Load()
				if current <= previousPeak || peak.CompareAndSwap(previousPeak, current) {
					break
				}
			}

			admitted <- struct{}{}
			<-release

			active.Add(-1)
			limiter.ReleaseSlot()
		})
	}

	close(start)

	for range maxConcurrency {
		select {
		case <-admitted:
		case <-time.After(time.Second):
			t.Fatal("timed out waiting for concurrent acquisitions")
		}
	}

	close(release)
	wg.Wait()

	if got := int(peak.Load()); got > maxConcurrency {
		t.Fatalf("peak admitted operations = %d, want <= %d", got, maxConcurrency)
	}
}
