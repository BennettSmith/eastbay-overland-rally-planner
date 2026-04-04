package trips_test

import (
	"testing"
	"time"

	"github.com/Overland-East-Bay/trip-planner-api/internal/app/trips"
)

func TestOptional_TriStateSemantics(t *testing.T) {
	t.Parallel()

	u := trips.Unspecified[string]()
	if u.IsSpecified() {
		t.Fatalf("Unspecified should not be specified")
	}
	if u.IsNull() {
		t.Fatalf("Unspecified should not be null")
	}
	if u.Value() != "" {
		t.Fatalf("Unspecified Value should be zero value")
	}

	n := trips.Null[string]()
	if !n.IsSpecified() {
		t.Fatalf("Null should be specified")
	}
	if !n.IsNull() {
		t.Fatalf("Null should be null")
	}

	s := trips.Some("x")
	if !s.IsSpecified() {
		t.Fatalf("Some should be specified")
	}
	if s.IsNull() {
		t.Fatalf("Some should not be null")
	}
	if s.Value() != "x" {
		t.Fatalf("Value=%q want=%q", s.Value(), "x")
	}
}

func TestOptional_TimeValue(t *testing.T) {
	t.Parallel()

	ts := time.Unix(123, 0).UTC()
	o := trips.Some(ts)
	if !o.IsSpecified() || o.IsNull() || !o.Value().Equal(ts) {
		t.Fatalf("o=%+v", o)
	}
}
