package rsvprepo

import (
	"context"
	"testing"
	"time"

	"github.com/Overland-East-Bay/trip-planner-api/internal/domain"
	"github.com/Overland-East-Bay/trip-planner-api/internal/ports/out/rsvprepo"
)

func TestRepo_GetUpsertCountYesList(t *testing.T) {
	t.Parallel()

	r := NewRepo()
	tripID := domain.TripID("t1")

	_, err := r.Get(context.Background(), tripID, "m1")
	if err != rsvprepo.ErrNotFound {
		t.Fatalf("Get(nonexistent) err=%v, want %v", err, rsvprepo.ErrNotFound)
	}

	t1 := time.Unix(10, 0).UTC()
	t2 := time.Unix(20, 0).UTC()

	if err := r.Upsert(context.Background(), rsvprepo.RSVP{TripID: tripID, MemberID: "m2", Status: rsvprepo.StatusNo, UpdatedAt: t2}); err != nil {
		t.Fatalf("Upsert(m2) err=%v", err)
	}
	if err := r.Upsert(context.Background(), rsvprepo.RSVP{TripID: tripID, MemberID: "m1", Status: rsvprepo.StatusYes, UpdatedAt: t1}); err != nil {
		t.Fatalf("Upsert(m1) err=%v", err)
	}

	got, err := r.Get(context.Background(), tripID, "m1")
	if err != nil {
		t.Fatalf("Get(m1) err=%v", err)
	}
	if got.Status != rsvprepo.StatusYes {
		t.Fatalf("Get(m1).Status=%q, want %q", got.Status, rsvprepo.StatusYes)
	}

	nYes, err := r.CountYesByTrip(context.Background(), tripID)
	if err != nil {
		t.Fatalf("CountYesByTrip() err=%v", err)
	}
	if nYes != 1 {
		t.Fatalf("CountYesByTrip()=%d, want 1", nYes)
	}

	list, err := r.ListByTrip(context.Background(), tripID)
	if err != nil {
		t.Fatalf("ListByTrip() err=%v", err)
	}
	if len(list) != 2 {
		t.Fatalf("ListByTrip() len=%d, want 2", len(list))
	}
	// Ordered by memberID ascending.
	if list[0].MemberID != "m1" || list[1].MemberID != "m2" {
		t.Fatalf("ListByTrip() order=%v, want [m1 m2]", []domain.MemberID{list[0].MemberID, list[1].MemberID})
	}
}

func TestRepo_ListByMember_OrdersByTripIDThenUpdatedAt(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	r := NewRepo()

	t1 := time.Unix(10, 0).UTC()
	t2 := time.Unix(20, 0).UTC()
	t3 := time.Unix(30, 0).UTC()

	_ = r.Upsert(ctx, rsvprepo.RSVP{TripID: "t2", MemberID: "m1", Status: rsvprepo.StatusNo, UpdatedAt: t2})
	_ = r.Upsert(ctx, rsvprepo.RSVP{TripID: "t1", MemberID: "m1", Status: rsvprepo.StatusYes, UpdatedAt: t3})
	_ = r.Upsert(ctx, rsvprepo.RSVP{TripID: "t1", MemberID: "m1", Status: rsvprepo.StatusNo, UpdatedAt: t1}) // overwrite same key; updatedAt changes ordering
	_ = r.Upsert(ctx, rsvprepo.RSVP{TripID: "t1", MemberID: "m2", Status: rsvprepo.StatusYes, UpdatedAt: t1})

	list, err := r.ListByMember(ctx, "m1")
	if err != nil {
		t.Fatalf("ListByMember: %v", err)
	}
	if len(list) != 2 {
		t.Fatalf("len=%d want=2", len(list))
	}
	// TripID order first: t1 then t2.
	if list[0].TripID != "t1" || list[1].TripID != "t2" {
		t.Fatalf("order=%v", []domain.TripID{list[0].TripID, list[1].TripID})
	}
}

func TestRepo_DeleteByMember_DeletesAllForMember(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	r := NewRepo()

	_ = r.Upsert(ctx, rsvprepo.RSVP{TripID: "t1", MemberID: "m1", Status: rsvprepo.StatusYes, UpdatedAt: time.Unix(1, 0).UTC()})
	_ = r.Upsert(ctx, rsvprepo.RSVP{TripID: "t2", MemberID: "m1", Status: rsvprepo.StatusNo, UpdatedAt: time.Unix(2, 0).UTC()})
	_ = r.Upsert(ctx, rsvprepo.RSVP{TripID: "t1", MemberID: "m2", Status: rsvprepo.StatusYes, UpdatedAt: time.Unix(3, 0).UTC()})

	if err := r.DeleteByMember(ctx, "m1"); err != nil {
		t.Fatalf("DeleteByMember: %v", err)
	}

	if _, err := r.Get(ctx, "t1", "m1"); err != rsvprepo.ErrNotFound {
		t.Fatalf("expected deleted: %v", err)
	}
	if _, err := r.Get(ctx, "t2", "m1"); err != rsvprepo.ErrNotFound {
		t.Fatalf("expected deleted: %v", err)
	}
	// Other member remains.
	if _, err := r.Get(ctx, "t1", "m2"); err != nil {
		t.Fatalf("expected m2 record present: %v", err)
	}
}
