package trips_test

import (
	"context"
	"errors"
	"testing"
	"time"

	memmemberrepo "github.com/Overland-East-Bay/trip-planner-api/internal/adapters/memory/memberrepo"
	memrsvprepo "github.com/Overland-East-Bay/trip-planner-api/internal/adapters/memory/rsvprepo"
	memtriprepo "github.com/Overland-East-Bay/trip-planner-api/internal/adapters/memory/triprepo"
	"github.com/Overland-East-Bay/trip-planner-api/internal/app/trips"
	"github.com/Overland-East-Bay/trip-planner-api/internal/domain"
	portrsvprepo "github.com/Overland-East-Bay/trip-planner-api/internal/ports/out/rsvprepo"
	porttriprepo "github.com/Overland-East-Bay/trip-planner-api/internal/ports/out/triprepo"
)

func TestService_CreateTripDraft_RejectsMissingCallerAndEmptyName(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	membersRepo := memmemberrepo.NewRepo()
	tripsRepo := memtriprepo.NewRepo()
	rsvpsRepo := memrsvprepo.NewRepo()
	svc := trips.NewService(tripsRepo, membersRepo, rsvpsRepo)

	// Missing caller member.
	_, err := svc.CreateTripDraft(ctx, "missing", trips.CreateTripDraftInput{Name: "Trip"})
	if err == nil {
		t.Fatalf("expected error")
	}
	ae := (*trips.Error)(nil)
	if !errors.As(err, &ae) || ae.Status != 422 {
		t.Fatalf("err=%v", err)
	}

	// Empty name after normalization.
	provisionMember(t, membersRepo, "m1")
	_, err = svc.CreateTripDraft(ctx, "m1", trips.CreateTripDraftInput{Name: "   \n\t"})
	if err == nil {
		t.Fatalf("expected error")
	}
	if !errors.As(err, &ae) || ae.Status != 422 {
		t.Fatalf("err=%v", err)
	}
}

func TestService_SetMyRSVP_RejectsInvalidResponse(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	membersRepo := memmemberrepo.NewRepo()
	tripsRepo := memtriprepo.NewRepo()
	rsvpsRepo := memrsvprepo.NewRepo()
	provisionMember(t, membersRepo, "m1")

	svc := trips.NewService(tripsRepo, membersRepo, rsvpsRepo)

	name := "Trip"
	cap := 1
	att := 0
	now := time.Unix(1, 0).UTC()
	_ = tripsRepo.Create(ctx, porttriprepo.Trip{
		ID:                 "tp",
		Status:             porttriprepo.StatusPublished,
		Name:               &name,
		CapacityRigs:       &cap,
		AttendingRigs:      &att,
		CreatorMemberID:    "m1",
		OrganizerMemberIDs: []domain.MemberID{"m1"},
		DraftVisibility:    porttriprepo.DraftVisibilityPublic,
		CreatedAt:          now,
		UpdatedAt:          now,
	})

	_, err := svc.SetMyRSVP(ctx, "m1", "tp", domain.RSVPResponse("MAYBE"))
	if err == nil {
		t.Fatalf("expected error")
	}
	ae := (*trips.Error)(nil)
	if !errors.As(err, &ae) || ae.Status != 422 {
		t.Fatalf("err=%v", err)
	}
}

func TestService_GetTripRSVPSummary_DraftNotAvailable(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	membersRepo := memmemberrepo.NewRepo()
	tripsRepo := memtriprepo.NewRepo()
	rsvpsRepo := memrsvprepo.NewRepo()
	provisionMember(t, membersRepo, "m1")

	svc := trips.NewService(tripsRepo, membersRepo, rsvpsRepo)

	name := "Draft"
	now := time.Unix(2, 0).UTC()
	_ = tripsRepo.Create(ctx, porttriprepo.Trip{
		ID:                 "td",
		Status:             porttriprepo.StatusDraft,
		Name:               &name,
		CreatorMemberID:    "m1",
		OrganizerMemberIDs: []domain.MemberID{"m1"},
		DraftVisibility:    porttriprepo.DraftVisibilityPrivate,
		CreatedAt:          now,
		UpdatedAt:          now,
	})

	_, err := svc.GetTripRSVPSummary(ctx, "m1", "td")
	if err == nil {
		t.Fatalf("expected error")
	}
	ae := (*trips.Error)(nil)
	if !errors.As(err, &ae) || ae.Status != 409 || ae.Code != "RSVP_NOT_AVAILABLE" {
		t.Fatalf("err=%v", err)
	}
}

func TestService_UpdateTrip_ValidatesCapacityAndPublishedInvariant(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	membersRepo := memmemberrepo.NewRepo()
	tripsRepo := memtriprepo.NewRepo()
	rsvpsRepo := memrsvprepo.NewRepo()
	provisionMember(t, membersRepo, "m1")

	svc := trips.NewService(tripsRepo, membersRepo, rsvpsRepo)

	name := "Trip"
	cap := 5
	att := 4
	now := time.Unix(3, 0).UTC()
	_ = tripsRepo.Create(ctx, porttriprepo.Trip{
		ID:                 "tp",
		Status:             porttriprepo.StatusPublished,
		Name:               &name,
		CapacityRigs:       &cap,
		AttendingRigs:      &att,
		CreatorMemberID:    "m1",
		OrganizerMemberIDs: []domain.MemberID{"m1"},
		DraftVisibility:    porttriprepo.DraftVisibilityPublic,
		CreatedAt:          now,
		UpdatedAt:          now,
	})

	// Capacity must be >= 1.
	_, err := svc.UpdateTrip(ctx, "m1", "tp", trips.UpdateTripInput{
		CapacityRigs: trips.Some(0),
	})
	if err == nil {
		t.Fatalf("expected error")
	}
	ae := (*trips.Error)(nil)
	if !errors.As(err, &ae) || ae.Status != 422 {
		t.Fatalf("err=%v", err)
	}

	// Cannot reduce below attending rigs.
	_, err = svc.UpdateTrip(ctx, "m1", "tp", trips.UpdateTripInput{
		CapacityRigs: trips.Some(3),
	})
	if err == nil {
		t.Fatalf("expected error")
	}
	if !errors.As(err, &ae) || ae.Status != 409 || ae.Code != "CAPACITY_BELOW_ATTENDANCE" {
		t.Fatalf("err=%v", err)
	}
}

func TestService_AddTripOrganizer_RejectsUnknownTargetMember(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	membersRepo := memmemberrepo.NewRepo()
	tripsRepo := memtriprepo.NewRepo()
	rsvpsRepo := memrsvprepo.NewRepo()
	provisionMember(t, membersRepo, "m1")

	svc := trips.NewService(tripsRepo, membersRepo, rsvpsRepo)

	name := "Trip"
	now := time.Unix(4, 0).UTC()
	_ = tripsRepo.Create(ctx, porttriprepo.Trip{
		ID:                 "tp",
		Status:             porttriprepo.StatusDraft,
		Name:               &name,
		CreatorMemberID:    "m1",
		OrganizerMemberIDs: []domain.MemberID{"m1"},
		DraftVisibility:    porttriprepo.DraftVisibilityPublic,
		CreatedAt:          now,
		UpdatedAt:          now,
	})

	_, err := svc.AddTripOrganizer(ctx, "m1", "tp", "missing")
	if err == nil {
		t.Fatalf("expected error")
	}
	ae := (*trips.Error)(nil)
	if !errors.As(err, &ae) || ae.Status != 422 {
		t.Fatalf("err=%v", err)
	}
}

func TestService_DeleteAllRSVPsByMember_UpdatesAttendanceUsingRepoCounts(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	membersRepo := memmemberrepo.NewRepo()
	tripsRepo := memtriprepo.NewRepo()
	rsvpsRepo := memrsvprepo.NewRepo()
	provisionMember(t, membersRepo, "m1")
	provisionMember(t, membersRepo, "m2")

	svc := trips.NewService(tripsRepo, membersRepo, rsvpsRepo)

	name := "Trip"
	cap := 10
	att := 0
	now := time.Unix(5, 0).UTC()
	_ = tripsRepo.Create(ctx, porttriprepo.Trip{
		ID:                 "t1",
		Status:             porttriprepo.StatusPublished,
		Name:               &name,
		CapacityRigs:       &cap,
		AttendingRigs:      &att,
		CreatorMemberID:    "m1",
		OrganizerMemberIDs: []domain.MemberID{"m1"},
		DraftVisibility:    porttriprepo.DraftVisibilityPublic,
		CreatedAt:          now,
		UpdatedAt:          now,
	})

	_ = rsvpsRepo.Upsert(ctx, portrsvprepo.RSVP{TripID: "t1", MemberID: "m1", Status: portrsvprepo.StatusYes, UpdatedAt: now})
	_ = rsvpsRepo.Upsert(ctx, portrsvprepo.RSVP{TripID: "t1", MemberID: "m2", Status: portrsvprepo.StatusYes, UpdatedAt: now})

	if err := svc.DeleteAllRSVPsByMember(ctx, "m1"); err != nil {
		t.Fatalf("DeleteAllRSVPsByMember: %v", err)
	}

	// Recomputed attendance should now be 1 (m2 only).
	t1, _ := tripsRepo.GetByID(ctx, "t1")
	if t1.AttendingRigs == nil || *t1.AttendingRigs != 1 {
		t.Fatalf("attending=%v want=1", t1.AttendingRigs)
	}
}

func TestService_NewService_DefaultTripIDGeneratorProducesNonEmptyID(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	membersRepo := memmemberrepo.NewRepo()
	tripsRepo := memtriprepo.NewRepo()
	rsvpsRepo := memrsvprepo.NewRepo()
	provisionMember(t, membersRepo, "m1")

	svc := trips.NewService(tripsRepo, membersRepo, rsvpsRepo)
	created, err := svc.CreateTripDraft(ctx, "m1", trips.CreateTripDraftInput{Name: "Trip"})
	if err != nil {
		t.Fatalf("CreateTripDraft: %v", err)
	}
	if created.ID == "" {
		t.Fatalf("expected non-empty id")
	}
}

func TestService_GetTripRSVPSummary_NotVisibleReturns404(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	membersRepo := memmemberrepo.NewRepo()
	tripsRepo := memtriprepo.NewRepo()
	rsvpsRepo := memrsvprepo.NewRepo()
	provisionMember(t, membersRepo, "m1")
	provisionMember(t, membersRepo, "m2")

	svc := trips.NewService(tripsRepo, membersRepo, rsvpsRepo)

	name := "Draft"
	now := time.Unix(6, 0).UTC()
	_ = tripsRepo.Create(ctx, porttriprepo.Trip{
		ID:                 "td-private",
		Status:             porttriprepo.StatusDraft,
		Name:               &name,
		CreatorMemberID:    "m2",
		OrganizerMemberIDs: []domain.MemberID{"m2"},
		DraftVisibility:    porttriprepo.DraftVisibilityPrivate,
		CreatedAt:          now,
		UpdatedAt:          now,
	})

	_, err := svc.GetTripRSVPSummary(ctx, "m1", "td-private")
	if err == nil {
		t.Fatalf("expected error")
	}
	ae := (*trips.Error)(nil)
	if !errors.As(err, &ae) || ae.Status != 404 {
		t.Fatalf("err=%v", err)
	}
}

func TestService_UpdateTrip_ClearsMeetingLocationWithNull(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	membersRepo := memmemberrepo.NewRepo()
	tripsRepo := memtriprepo.NewRepo()
	rsvpsRepo := memrsvprepo.NewRepo()
	provisionMember(t, membersRepo, "m1")

	svc := trips.NewService(tripsRepo, membersRepo, rsvpsRepo)

	name := "Trip"
	now := time.Unix(7, 0).UTC()
	addr := "123 Main"
	_ = tripsRepo.Create(ctx, porttriprepo.Trip{
		ID:                 "tp",
		Status:             porttriprepo.StatusDraft,
		Name:               &name,
		CreatorMemberID:    "m1",
		OrganizerMemberIDs: []domain.MemberID{"m1"},
		DraftVisibility:    porttriprepo.DraftVisibilityPublic,
		MeetingLocation:    &domain.Location{Label: "Meet", Address: &addr},
		CreatedAt:          now,
		UpdatedAt:          now,
	})

	td, err := svc.UpdateTrip(ctx, "m1", "tp", trips.UpdateTripInput{
		MeetingLocation: trips.Null[*trips.LocationPatch](),
	})
	if err != nil {
		t.Fatalf("UpdateTrip: %v", err)
	}
	if td.MeetingLocation != nil {
		t.Fatalf("expected meetingLocation cleared")
	}
}

func TestService_CancelTrip_NonVisibleStatusReturnsNotFound(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	membersRepo := memmemberrepo.NewRepo()
	tripsRepo := memtriprepo.NewRepo()
	rsvpsRepo := memrsvprepo.NewRepo()
	provisionMember(t, membersRepo, "m1")

	svc := trips.NewService(tripsRepo, membersRepo, rsvpsRepo)

	name := "Trip"
	now := time.Unix(8, 0).UTC()
	_ = tripsRepo.Create(ctx, porttriprepo.Trip{
		ID:                 "tp",
		Status:             porttriprepo.Status("WEIRD"),
		Name:               &name,
		CreatorMemberID:    "m1",
		OrganizerMemberIDs: []domain.MemberID{"m1"},
		DraftVisibility:    porttriprepo.DraftVisibilityPublic,
		CreatedAt:          now,
		UpdatedAt:          now,
	})

	_, err := svc.CancelTrip(ctx, "m1", "tp")
	if err == nil {
		t.Fatalf("expected error")
	}
	ae := (*trips.Error)(nil)
	// Non-visible trips should fail closed as 404 (even if they exist).
	if !errors.As(err, &ae) || ae.Status != 404 || ae.Code != "TRIP_NOT_FOUND" {
		t.Fatalf("err=%v", err)
	}
}

func TestService_UpdateTrip_DraftAuthorizationAndNameValidation(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	membersRepo := memmemberrepo.NewRepo()
	tripsRepo := memtriprepo.NewRepo()
	rsvpsRepo := memrsvprepo.NewRepo()
	provisionMember(t, membersRepo, "m1")
	provisionMember(t, membersRepo, "m2")

	svc := trips.NewService(tripsRepo, membersRepo, rsvpsRepo)

	name := "Trip"
	now := time.Unix(9, 0).UTC()
	_ = tripsRepo.Create(ctx, porttriprepo.Trip{
		ID:                 "td-priv",
		Status:             porttriprepo.StatusDraft,
		Name:               &name,
		CreatorMemberID:    "m1",
		OrganizerMemberIDs: []domain.MemberID{"m1"},
		DraftVisibility:    porttriprepo.DraftVisibilityPrivate,
		CreatedAt:          now,
		UpdatedAt:          now,
	})
	_ = tripsRepo.Create(ctx, porttriprepo.Trip{
		ID:                 "td-pub",
		Status:             porttriprepo.StatusDraft,
		Name:               &name,
		CreatorMemberID:    "m1",
		OrganizerMemberIDs: []domain.MemberID{"m1"},
		DraftVisibility:    porttriprepo.DraftVisibilityPublic,
		CreatedAt:          now,
		UpdatedAt:          now,
	})

	// Private draft: only creator may update.
	_, err := svc.UpdateTrip(ctx, "m2", "td-priv", trips.UpdateTripInput{Name: trips.Some("New")})
	if err == nil {
		t.Fatalf("expected error")
	}
	ae := (*trips.Error)(nil)
	if !errors.As(err, &ae) || ae.Status != 404 {
		t.Fatalf("err=%v", err)
	}

	// Public draft: only organizers may update.
	_, err = svc.UpdateTrip(ctx, "m2", "td-pub", trips.UpdateTripInput{Name: trips.Some("New")})
	if err == nil {
		t.Fatalf("expected error")
	}
	if !errors.As(err, &ae) || ae.Status != 404 {
		t.Fatalf("err=%v", err)
	}

	// Name cannot be null.
	_, err = svc.UpdateTrip(ctx, "m1", "td-priv", trips.UpdateTripInput{Name: trips.Null[string]()})
	if err == nil {
		t.Fatalf("expected error")
	}
	if !errors.As(err, &ae) || ae.Status != 422 {
		t.Fatalf("err=%v", err)
	}
}

func TestService_UpdateTrip_CanceledCannotBeModified(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	membersRepo := memmemberrepo.NewRepo()
	tripsRepo := memtriprepo.NewRepo()
	rsvpsRepo := memrsvprepo.NewRepo()
	provisionMember(t, membersRepo, "m1")

	svc := trips.NewService(tripsRepo, membersRepo, rsvpsRepo)

	name := "Trip"
	now := time.Unix(10, 0).UTC()
	_ = tripsRepo.Create(ctx, porttriprepo.Trip{
		ID:                 "tc",
		Status:             porttriprepo.StatusCanceled,
		Name:               &name,
		CreatorMemberID:    "m1",
		OrganizerMemberIDs: []domain.MemberID{"m1"},
		DraftVisibility:    porttriprepo.DraftVisibilityPublic,
		CreatedAt:          now,
		UpdatedAt:          now,
	})

	_, err := svc.UpdateTrip(ctx, "m1", "tc", trips.UpdateTripInput{Name: trips.Some("New")})
	if err == nil {
		t.Fatalf("expected error")
	}
	ae := (*trips.Error)(nil)
	if !errors.As(err, &ae) || ae.Status != 409 || ae.Code != "TRIP_CANCELED" {
		t.Fatalf("err=%v", err)
	}
}

func TestService_CreateTripDraft_IDConflictReturns409(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	membersRepo := memmemberrepo.NewRepo()
	tripsRepo := memtriprepo.NewRepo()
	rsvpsRepo := memrsvprepo.NewRepo()
	provisionMember(t, membersRepo, "m1")

	svc := trips.NewService(tripsRepo, membersRepo, rsvpsRepo)
	svc.SetNewTripIDForTest(func() domain.TripID { return "fixed-id" })

	if _, err := svc.CreateTripDraft(ctx, "m1", trips.CreateTripDraftInput{Name: "Trip"}); err != nil {
		t.Fatalf("CreateTripDraft: %v", err)
	}
	_, err := svc.CreateTripDraft(ctx, "m1", trips.CreateTripDraftInput{Name: "Trip"})
	if err == nil {
		t.Fatalf("expected error")
	}
	ae := (*trips.Error)(nil)
	if !errors.As(err, &ae) || ae.Status != 409 || ae.Code != "TRIP_ID_CONFLICT" {
		t.Fatalf("err=%v", err)
	}
}
