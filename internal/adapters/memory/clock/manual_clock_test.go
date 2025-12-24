package clock

import (
	"sync"
	"testing"
	"time"
)

func TestManualClock_NowSetAdd(t *testing.T) {
	t.Parallel()

	start := time.Date(2025, 12, 24, 0, 0, 0, 0, time.UTC)
	c := NewManualClock(start)

	if got := c.Now(); !got.Equal(start) {
		t.Fatalf("Now()=%v, want %v", got, start)
	}

	next := start.Add(10 * time.Minute)
	c.Set(next)
	if got := c.Now(); !got.Equal(next) {
		t.Fatalf("Now() after Set()=%v, want %v", got, next)
	}

	c.Add(5 * time.Second)
	if got := c.Now(); !got.Equal(next.Add(5 * time.Second)) {
		t.Fatalf("Now() after Add()=%v, want %v", got, next.Add(5*time.Second))
	}
}

func TestManualClock_ConcurrentNow(t *testing.T) {
	t.Parallel()

	c := NewManualClock(time.Unix(0, 0).UTC())

	var wg sync.WaitGroup
	for i := 0; i < 20; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_ = c.Now()
		}()
	}
	wg.Wait()
}
