package clock

import (
	"testing"
	"time"
)

func TestSystemClock_Now_IsUTCAndWithinWallClockBounds(t *testing.T) {
	t.Parallel()

	c := NewSystemClock()

	before := time.Now().UTC()
	got := c.Now()
	after := time.Now().UTC()

	if got.IsZero() {
		t.Fatalf("Now returned zero time")
	}
	if got.Location() != time.UTC {
		t.Fatalf("location=%v want=UTC", got.Location())
	}
	if got.Before(before) || got.After(after) {
		t.Fatalf("got=%s not within [%s, %s]", got, before, after)
	}
}
