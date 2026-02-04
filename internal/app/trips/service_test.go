package trips_test

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	memmemberrepo "github.com/Overland-East-Bay/trip-planner-api/internal/adapters/memory/memberrepo"
	memrsvprepo "github.com/Overland-East-Bay/trip-planner-api/internal/adapters/memory/rsvprepo"
	memtriprepo "github.com/Overland-East-Bay/trip-planner-api/internal/adapters/memory/triprepo"
	"github.com/Overland-East-Bay/trip-planner-api/internal/app/trips"
	"github.com/Overland-East-Bay/trip-planner-api/internal/domain"
	portmemberrepo "github.com/Overland-East-Bay/trip-planner-api/internal/ports/out/memberrepo"
	portrsvprepo "github.com/Overland-East-Bay/trip-planner-api/internal/ports/out/rsvprepo"
	porttriprepo "github.com/Overland-East-Bay/trip-planner-api/internal/ports/out/triprepo"
)

func provisionMember(t *testing.T, repo *memmemberrepo.Repo, id domain.MemberID) {
	t.Helper()
	now := time.Unix(100, 0).UTC()
	if err := repo.Create(context.Background(), portmemberrepo.Member{
		ID:          id,
		Subject:     domain.SubjectID("sub-" + string(id)),
		DisplayName: "Member " + string(id),
		Email:       string(id) + "@example.com",
		IsActive:    true,
		CreatedAt:   now,
		UpdatedAt:   now,
	}); err != nil {
		t.Fatalf("create member: %v", err)
	}
}

func TestService_CreateTripDraft_NormalizesAndSetsFields(t *testing.T) {
	t.Parallel()

	membersRepo := memmemberrepo.NewRepo()
	tripsRepo := memtriprepo.NewRepo()
	rsvpsRepo := memrsvprepo.NewRepo()
	provisionMember(t, membersRepo, "m1")

	svc := trips.NewService(tripsRepo, membersRepo, rsvpsRepo)
	svc.SetNewTripIDForTest(func() domain.TripID { return "t1" })

	created, err := svc.CreateTripDraft(context.Background(), "m1", trips.CreateTripDraftInput{Name: "  Snow   Run  "})
	if err != nil {
		t.Fatalf("CreateTripDraft: %v", err)
	}
	if created.ID != "t1" || created.Status != domain.TripStatusDraft || created.DraftVisibility != domain.DraftVisibilityPrivate {
		t.Fatalf("created=%+v", created)
	}

	tp, err := tripsRepo.GetByID(context.Background(), "t1")
	if err != nil {
		t.Fatalf("GetByID: %v", err)
	}
	if tp.Status != porttriprepo.StatusDraft || tp.DraftVisibility != porttriprepo.DraftVisibilityPrivate {
		t.Fatalf("status/dv=%s/%s", tp.Status, tp.DraftVisibility)
	}
	if tp.CreatorMemberID != "m1" {
		t.Fatalf("creator=%s", tp.CreatorMemberID)
	}
	if len(tp.OrganizerMemberIDs) != 1 || tp.OrganizerMemberIDs[0] != "m1" {
		t.Fatalf("organizers=%v", tp.OrganizerMemberIDs)
	}
	if tp.Name == nil || *tp.Name != "Snow Run" {
		t.Fatalf("name=%v", tp.Name)
	}
}

func TestService_UpdateTrip_DraftVisibilityAuthz(t *testing.T) {
	t.Parallel()

	membersRepo := memmemberrepo.NewRepo()
	tripsRepo := memtriprepo.NewRepo()
	rsvpsRepo := memrsvprepo.NewRepo()
	provisionMember(t, membersRepo, "m1")
	provisionMember(t, membersRepo, "m2")

	svc := trips.NewService(tripsRepo, membersRepo, rsvpsRepo)

	name := "Draft"
	now := time.Unix(200, 0).UTC()
	_ = tripsRepo.Create(context.Background(), porttriprepo.Trip{
		ID:                 "td1",
		Status:             porttriprepo.StatusDraft,
		Name:               &name,
		CreatorMemberID:    "m1",
		OrganizerMemberIDs: []domain.MemberID{"m1"},
		DraftVisibility:    porttriprepo.DraftVisibilityPrivate,
		CreatedAt:          now,
		UpdatedAt:          now,
	})

	_, err := svc.UpdateTrip(context.Background(), "m2", "td1", trips.UpdateTripInput{Name: trips.Some("X")})
	if err == nil {
		t.Fatalf("expected error")
	}
	var ae *trips.Error
	if !errors.As(err, &ae) || ae.Status != 404 {
		t.Fatalf("err=%v", err)
	}
}

func TestService_PublishTrip_RequiresPublicDraftAndRequiredFields(t *testing.T) {
	t.Parallel()

	membersRepo := memmemberrepo.NewRepo()
	tripsRepo := memtriprepo.NewRepo()
	rsvpsRepo := memrsvprepo.NewRepo()
	provisionMember(t, membersRepo, "m1")

	svc := trips.NewService(tripsRepo, membersRepo, rsvpsRepo)

	name := "Trip"
	now := time.Unix(300, 0).UTC()
	_ = tripsRepo.Create(context.Background(), porttriprepo.Trip{
		ID:                 "tpub",
		Status:             porttriprepo.StatusDraft,
		Name:               &name,
		CreatorMemberID:    "m1",
		OrganizerMemberIDs: []domain.MemberID{"m1"},
		DraftVisibility:    porttriprepo.DraftVisibilityPrivate,
		CreatedAt:          now,
		UpdatedAt:          now,
	})

	_, _, err := svc.PublishTrip(context.Background(), "m1", "tpub")
	if err == nil {
		t.Fatalf("expected error")
	}
	var ae *trips.Error
	if !errors.As(err, &ae) || ae.Status != 409 {
		t.Fatalf("err=%v", err)
	}

	// Make it public, but still missing required publish fields.
	tp, _ := tripsRepo.GetByID(context.Background(), "tpub")
	tp.DraftVisibility = porttriprepo.DraftVisibilityPublic
	_ = tripsRepo.Save(context.Background(), tp)

	_, _, err = svc.PublishTrip(context.Background(), "m1", "tpub")
	if err == nil {
		t.Fatalf("expected error")
	}
	if !errors.As(err, &ae) || ae.Status != 409 || ae.Code != "TRIP_NOT_READY_TO_PUBLISH" {
		t.Fatalf("err=%v", err)
	}
}

func TestService_CancelTrip_IdempotentAndLocksFurtherUpdates(t *testing.T) {
	t.Parallel()

	membersRepo := memmemberrepo.NewRepo()
	tripsRepo := memtriprepo.NewRepo()
	rsvpsRepo := memrsvprepo.NewRepo()
	provisionMember(t, membersRepo, "m1")

	svc := trips.NewService(tripsRepo, membersRepo, rsvpsRepo)

	name := "Trip"
	now := time.Unix(400, 0).UTC()
	_ = tripsRepo.Create(context.Background(), porttriprepo.Trip{
		ID:                 "tc",
		Status:             porttriprepo.StatusPublished,
		Name:               &name,
		CreatorMemberID:    "m1",
		OrganizerMemberIDs: []domain.MemberID{"m1"},
		DraftVisibility:    porttriprepo.DraftVisibilityPublic,
		CreatedAt:          now,
		UpdatedAt:          now,
	})

	td, err := svc.CancelTrip(context.Background(), "m1", "tc")
	if err != nil {
		t.Fatalf("CancelTrip: %v", err)
	}
	if td.Status != domain.TripStatusCanceled {
		t.Fatalf("status=%s", td.Status)
	}

	// Idempotent.
	td2, err := svc.CancelTrip(context.Background(), "m1", "tc")
	if err != nil {
		t.Fatalf("CancelTrip2: %v", err)
	}
	if td2.Status != domain.TripStatusCanceled {
		t.Fatalf("status2=%s", td2.Status)
	}

	_, err = svc.UpdateTrip(context.Background(), "m1", "tc", trips.UpdateTripInput{Name: trips.Some("New")})
	if err == nil {
		t.Fatalf("expected error")
	}
	var ae *trips.Error
	if !errors.As(err, &ae) || ae.Status != 409 {
		t.Fatalf("err=%v", err)
	}
}

func TestService_OrganizerManagement_AddRemoveAndLastOrganizerInvariant(t *testing.T) {
	t.Parallel()

	membersRepo := memmemberrepo.NewRepo()
	tripsRepo := memtriprepo.NewRepo()
	rsvpsRepo := memrsvprepo.NewRepo()
	provisionMember(t, membersRepo, "m1")
	provisionMember(t, membersRepo, "m2")

	svc := trips.NewService(tripsRepo, membersRepo, rsvpsRepo)

	name := "Trip"
	now := time.Unix(500, 0).UTC()
	_ = tripsRepo.Create(context.Background(), porttriprepo.Trip{
		ID:                 "to",
		Status:             porttriprepo.StatusPublished,
		Name:               &name,
		CreatorMemberID:    "m1",
		OrganizerMemberIDs: []domain.MemberID{"m1"},
		DraftVisibility:    porttriprepo.DraftVisibilityPublic,
		CreatedAt:          now,
		UpdatedAt:          now,
	})

	td, err := svc.AddTripOrganizer(context.Background(), "m1", "to", "m2")
	if err != nil {
		t.Fatalf("AddTripOrganizer: %v", err)
	}
	if len(td.Organizers) != 2 {
		t.Fatalf("organizers=%d", len(td.Organizers))
	}

	// Remove one organizer, then ensure we cannot remove the last remaining organizer.
	_, err = svc.RemoveTripOrganizer(context.Background(), "m1", "to", "m2")
	if err != nil {
		t.Fatalf("RemoveTripOrganizer(m2): %v", err)
	}
	_, err = svc.RemoveTripOrganizer(context.Background(), "m1", "to", "m1")
	if err == nil {
		t.Fatalf("expected error")
	}
	var ae *trips.Error
	if !errors.As(err, &ae) || ae.Status != 409 {
		t.Fatalf("err=%v", err)
	}
}

func TestService_RSVP_PublishedOnly_CapacityAndIdempotency(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	membersRepo := memmemberrepo.NewRepo()
	tripsRepo := memtriprepo.NewRepo()
	rsvpsRepo := memrsvprepo.NewRepo()
	provisionMember(t, membersRepo, "m1")
	provisionMember(t, membersRepo, "m2")

	svc := trips.NewService(tripsRepo, membersRepo, rsvpsRepo)

	name := "Trip"
	now := time.Unix(600, 0).UTC()
	cap := 1
	att0 := 0
	_ = tripsRepo.Create(ctx, porttriprepo.Trip{
		ID:                 "tp",
		Status:             porttriprepo.StatusPublished,
		Name:               &name,
		CapacityRigs:       &cap,
		AttendingRigs:      &att0,
		CreatorMemberID:    "m1",
		OrganizerMemberIDs: []domain.MemberID{"m1"},
		DraftVisibility:    porttriprepo.DraftVisibilityPublic,
		CreatedAt:          now,
		UpdatedAt:          now,
	})

	// First YES should succeed and consume capacity.
	my1, err := svc.SetMyRSVP(ctx, "m1", "tp", domain.RSVPResponseYes)
	if err != nil {
		t.Fatalf("SetMyRSVP(YES): %v", err)
	}
	if my1.Response != domain.RSVPResponseYes || my1.MemberID != "m1" || my1.TripID != "tp" || my1.UpdatedAt.IsZero() {
		t.Fatalf("my1=%+v", my1)
	}

	// Second YES should fail at capacity.
	_, err = svc.SetMyRSVP(ctx, "m2", "tp", domain.RSVPResponseYes)
	if err == nil {
		t.Fatalf("expected error")
	}
	var ae *trips.Error
	if !errors.As(err, &ae) || ae.Status != 409 || ae.Code != "TRIP_AT_CAPACITY" {
		t.Fatalf("err=%v", err)
	}

	// Changing from YES -> NO releases capacity.
	_, err = svc.SetMyRSVP(ctx, "m1", "tp", domain.RSVPResponseNo)
	if err != nil {
		t.Fatalf("SetMyRSVP(NO): %v", err)
	}
	// Now m2 can RSVP YES.
	_, err = svc.SetMyRSVP(ctx, "m2", "tp", domain.RSVPResponseYes)
	if err != nil {
		t.Fatalf("SetMyRSVP(m2 YES): %v", err)
	}

	// Idempotent no-op (same value) should preserve UpdatedAt.
	existing, _ := rsvpsRepo.Get(ctx, "tp", "m2")
	my2, err := svc.SetMyRSVP(ctx, "m2", "tp", domain.RSVPResponseYes)
	if err != nil {
		t.Fatalf("SetMyRSVP(idempotent): %v", err)
	}
	if !my2.UpdatedAt.Equal(existing.UpdatedAt) {
		t.Fatalf("UpdatedAt changed on idempotent set: got=%s want=%s", my2.UpdatedAt, existing.UpdatedAt)
	}
}

func TestService_RSVP_Summary_SortsAndOmitsUnset(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	membersRepo := memmemberrepo.NewRepo()
	tripsRepo := memtriprepo.NewRepo()
	rsvpsRepo := memrsvprepo.NewRepo()

	// Members with display names designed to test sorting.
	now := time.Unix(700, 0).UTC()
	_ = membersRepo.Create(ctx, portmemberrepo.Member{ID: "m1", Subject: "sub-m1", DisplayName: "Zoe", Email: "m1@example.com", IsActive: true, CreatedAt: now, UpdatedAt: now})
	_ = membersRepo.Create(ctx, portmemberrepo.Member{ID: "m2", Subject: "sub-m2", DisplayName: "alice", Email: "m2@example.com", IsActive: true, CreatedAt: now, UpdatedAt: now})
	_ = membersRepo.Create(ctx, portmemberrepo.Member{ID: "m3", Subject: "sub-m3", DisplayName: "Bob", Email: "m3@example.com", IsActive: true, CreatedAt: now, UpdatedAt: now})

	name := "Trip"
	cap := 5
	att0 := 0
	_ = tripsRepo.Create(ctx, porttriprepo.Trip{
		ID:                 "tp",
		Status:             porttriprepo.StatusPublished,
		Name:               &name,
		CapacityRigs:       &cap,
		AttendingRigs:      &att0,
		CreatorMemberID:    "m1",
		OrganizerMemberIDs: []domain.MemberID{"m1"},
		DraftVisibility:    porttriprepo.DraftVisibilityPublic,
		CreatedAt:          now,
		UpdatedAt:          now,
	})

	// Seed RSVPs: YES (m3), NO (m1), UNSET (m2) -> UNSET omitted.
	_ = rsvpsRepo.Upsert(ctx, portrsvprepo.RSVP{TripID: "tp", MemberID: "m3", Status: portrsvprepo.StatusYes, UpdatedAt: now})
	_ = rsvpsRepo.Upsert(ctx, portrsvprepo.RSVP{TripID: "tp", MemberID: "m1", Status: portrsvprepo.StatusNo, UpdatedAt: now})
	_ = rsvpsRepo.Upsert(ctx, portrsvprepo.RSVP{TripID: "tp", MemberID: "m2", Status: portrsvprepo.StatusUnset, UpdatedAt: now})

	svc := trips.NewService(tripsRepo, membersRepo, rsvpsRepo)
	sum, err := svc.GetTripRSVPSummary(ctx, "m1", "tp")
	if err != nil {
		t.Fatalf("GetTripRSVPSummary: %v", err)
	}
	if sum.AttendingRigs != 1 {
		t.Fatalf("AttendingRigs=%d want=1", sum.AttendingRigs)
	}
	if len(sum.AttendingMembers) != 1 || sum.AttendingMembers[0].ID != "m3" {
		t.Fatalf("AttendingMembers=%v", sum.AttendingMembers)
	}
	if len(sum.NotAttendingMembers) != 1 || sum.NotAttendingMembers[0].ID != "m1" {
		t.Fatalf("NotAttendingMembers=%v", sum.NotAttendingMembers)
	}
}

func TestService_DeleteAllRSVPsByMember_RecomputesAttendanceAndIgnoresMissingTrips(t *testing.T) {
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
	now := time.Unix(800, 0).UTC()
	att0 := 0

	_ = tripsRepo.Create(ctx, porttriprepo.Trip{
		ID:                 "t1",
		Status:             porttriprepo.StatusPublished,
		Name:               &name,
		CapacityRigs:       &cap,
		AttendingRigs:      &att0,
		CreatorMemberID:    "m1",
		OrganizerMemberIDs: []domain.MemberID{"m1"},
		DraftVisibility:    porttriprepo.DraftVisibilityPublic,
		CreatedAt:          now,
		UpdatedAt:          now,
	})
	_ = tripsRepo.Create(ctx, porttriprepo.Trip{
		ID:                 "t2",
		Status:             porttriprepo.StatusPublished,
		Name:               &name,
		CapacityRigs:       &cap,
		AttendingRigs:      &att0,
		CreatorMemberID:    "m1",
		OrganizerMemberIDs: []domain.MemberID{"m1"},
		DraftVisibility:    porttriprepo.DraftVisibilityPublic,
		CreatedAt:          now,
		UpdatedAt:          now,
	})

	// Seed RSVPs for m1 across two real trips, plus one missing trip (should be ignored).
	_ = rsvpsRepo.Upsert(ctx, portrsvprepo.RSVP{TripID: "t1", MemberID: "m1", Status: portrsvprepo.StatusYes, UpdatedAt: now})
	_ = rsvpsRepo.Upsert(ctx, portrsvprepo.RSVP{TripID: "t2", MemberID: "m1", Status: portrsvprepo.StatusNo, UpdatedAt: now})
	_ = rsvpsRepo.Upsert(ctx, portrsvprepo.RSVP{TripID: "missing", MemberID: "m1", Status: portrsvprepo.StatusYes, UpdatedAt: now})
	_ = rsvpsRepo.Upsert(ctx, portrsvprepo.RSVP{TripID: "t1", MemberID: "m2", Status: portrsvprepo.StatusYes, UpdatedAt: now})

	if err := svc.DeleteAllRSVPsByMember(ctx, "m1"); err != nil {
		t.Fatalf("DeleteAllRSVPsByMember: %v", err)
	}

	// m1 RSVPs should be gone.
	if _, err := rsvpsRepo.Get(ctx, "t1", "m1"); err == nil {
		t.Fatalf("expected m1 RSVP deleted for t1")
	}
	if _, err := rsvpsRepo.Get(ctx, "t2", "m1"); err == nil {
		t.Fatalf("expected m1 RSVP deleted for t2")
	}

	// Attendance should be recomputed: only m2 has YES on t1 => 1. t2 has no YES => 0.
	t1, _ := tripsRepo.GetByID(ctx, "t1")
	if t1.AttendingRigs == nil || *t1.AttendingRigs != 1 {
		t.Fatalf("t1.AttendingRigs=%v want=1", t1.AttendingRigs)
	}
	t2, _ := tripsRepo.GetByID(ctx, "t2")
	if t2.AttendingRigs == nil || *t2.AttendingRigs != 0 {
		t.Fatalf("t2.AttendingRigs=%v want=0", t2.AttendingRigs)
	}
}

func TestService_ListVisibleTripsForMember_MapsTripSummaries(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	membersRepo := memmemberrepo.NewRepo()
	tripsRepo := memtriprepo.NewRepo()
	rsvpsRepo := memrsvprepo.NewRepo()
	provisionMember(t, membersRepo, "m1")

	svc := trips.NewService(tripsRepo, membersRepo, rsvpsRepo)

	name1 := "One"
	name2 := "Two"
	now := time.Unix(900, 0).UTC()
	_ = tripsRepo.Create(ctx, porttriprepo.Trip{ID: "t1", Status: porttriprepo.StatusPublished, Name: &name1, CreatedAt: now, UpdatedAt: now})
	_ = tripsRepo.Create(ctx, porttriprepo.Trip{ID: "t2", Status: porttriprepo.StatusCanceled, Name: &name2, CreatedAt: now, UpdatedAt: now})
	_ = tripsRepo.Create(ctx, porttriprepo.Trip{ID: "td", Status: porttriprepo.StatusDraft, Name: &name2, DraftVisibility: porttriprepo.DraftVisibilityPrivate, CreatorMemberID: "m1", OrganizerMemberIDs: []domain.MemberID{"m1"}, CreatedAt: now, UpdatedAt: now})

	out, err := svc.ListVisibleTripsForMember(ctx, "m1")
	if err != nil {
		t.Fatalf("ListVisibleTripsForMember: %v", err)
	}
	if len(out) != 2 {
		t.Fatalf("len=%d want=2", len(out))
	}
	ids := []string{string(out[0].ID), string(out[1].ID)}
	if !containsAll(ids, []string{"t1", "t2"}) {
		t.Fatalf("ids=%v want contains t1,t2", ids)
	}
}

func TestService_ListMyDraftTrips_FiltersByVisibilityRules(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	membersRepo := memmemberrepo.NewRepo()
	tripsRepo := memtriprepo.NewRepo()
	rsvpsRepo := memrsvprepo.NewRepo()
	provisionMember(t, membersRepo, "m1")
	provisionMember(t, membersRepo, "m2")

	svc := trips.NewService(tripsRepo, membersRepo, rsvpsRepo)

	now := time.Unix(1000, 0).UTC()
	name := "Draft"
	_ = tripsRepo.Create(ctx, porttriprepo.Trip{
		ID:                 "pub-org",
		Status:             porttriprepo.StatusDraft,
		Name:               &name,
		DraftVisibility:    porttriprepo.DraftVisibilityPublic,
		OrganizerMemberIDs: []domain.MemberID{"m1"},
		CreatedAt:          now,
		UpdatedAt:          now,
	})
	_ = tripsRepo.Create(ctx, porttriprepo.Trip{
		ID:              "priv-creator",
		Status:          porttriprepo.StatusDraft,
		Name:            &name,
		DraftVisibility: porttriprepo.DraftVisibilityPrivate,
		CreatorMemberID: "m1",
		CreatedAt:       now,
		UpdatedAt:       now,
	})
	_ = tripsRepo.Create(ctx, porttriprepo.Trip{
		ID:                 "pub-not-org",
		Status:             porttriprepo.StatusDraft,
		Name:               &name,
		DraftVisibility:    porttriprepo.DraftVisibilityPublic,
		OrganizerMemberIDs: []domain.MemberID{"m2"},
		CreatedAt:          now,
		UpdatedAt:          now,
	})
	_ = tripsRepo.Create(ctx, porttriprepo.Trip{
		ID:              "priv-not-creator",
		Status:          porttriprepo.StatusDraft,
		Name:            &name,
		DraftVisibility: porttriprepo.DraftVisibilityPrivate,
		CreatorMemberID: "m2",
		CreatedAt:       now,
		UpdatedAt:       now,
	})

	out, err := svc.ListMyDraftTrips(ctx, "m1")
	if err != nil {
		t.Fatalf("ListMyDraftTrips: %v", err)
	}
	if len(out) != 2 {
		t.Fatalf("len=%d want=2", len(out))
	}
	ids := []string{string(out[0].ID), string(out[1].ID)}
	if !containsAll(ids, []string{"pub-org", "priv-creator"}) {
		t.Fatalf("ids=%v want contains pub-org,priv-creator", ids)
	}
}

func TestService_GetTripDetails_DraftOmitsRSVPFields(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	membersRepo := memmemberrepo.NewRepo()
	tripsRepo := memtriprepo.NewRepo()
	rsvpsRepo := memrsvprepo.NewRepo()
	provisionMember(t, membersRepo, "m1")

	svc := trips.NewService(tripsRepo, membersRepo, rsvpsRepo)

	name := "Draft"
	now := time.Unix(1100, 0).UTC()
	_ = tripsRepo.Create(ctx, porttriprepo.Trip{
		ID:                 "td1",
		Status:             porttriprepo.StatusDraft,
		Name:               &name,
		CreatorMemberID:    "m1",
		OrganizerMemberIDs: []domain.MemberID{"m1"},
		DraftVisibility:    porttriprepo.DraftVisibilityPrivate,
		CreatedAt:          now,
		UpdatedAt:          now,
	})

	td, err := svc.GetTripDetails(ctx, "m1", "td1")
	if err != nil {
		t.Fatalf("GetTripDetails: %v", err)
	}
	if td.Status != domain.TripStatusDraft {
		t.Fatalf("status=%s", td.Status)
	}
	if td.RSVPActionsEnabled {
		t.Fatalf("RSVPActionsEnabled should be false for drafts")
	}
	if td.RSVPSummary != nil || td.MyRSVP != nil {
		t.Fatalf("expected RSVP fields omitted for draft: %+v", td)
	}
}

func TestService_GetTripDetails_PublishedIncludesRSVPSummaryAndMyRSVPWhenPresent(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	membersRepo := memmemberrepo.NewRepo()
	tripsRepo := memtriprepo.NewRepo()
	rsvpsRepo := memrsvprepo.NewRepo()

	// Two members to populate summary + organizer expansion.
	now := time.Unix(1200, 0).UTC()
	_ = membersRepo.Create(ctx, portmemberrepo.Member{ID: "m1", Subject: "sub-m1", DisplayName: "Alice", Email: "m1@example.com", IsActive: true, CreatedAt: now, UpdatedAt: now})
	_ = membersRepo.Create(ctx, portmemberrepo.Member{ID: "m2", Subject: "sub-m2", DisplayName: "Bob", Email: "m2@example.com", IsActive: true, CreatedAt: now, UpdatedAt: now})

	svc := trips.NewService(tripsRepo, membersRepo, rsvpsRepo)

	name := "Trip"
	desc := " Desc "
	diff := "Easy"
	comms := "FRS"
	recs := "Spare tire"
	start := time.Date(2026, 3, 1, 0, 0, 0, 0, time.UTC)
	end := time.Date(2026, 3, 2, 0, 0, 0, 0, time.UTC)
	cap := 5
	att := 0
	addr := "123 Main"
	_ = tripsRepo.Create(ctx, porttriprepo.Trip{
		ID:                          "tp1",
		Status:                      porttriprepo.StatusPublished,
		Name:                        &name,
		Description:                 &desc,
		DifficultyText:              &diff,
		CommsRequirementsText:       &comms,
		RecommendedRequirementsText: &recs,
		StartDate:                   &start,
		EndDate:                     &end,
		CapacityRigs:                &cap,
		AttendingRigs:               &att,
		MeetingLocation:             &domain.Location{Label: "Meet", Address: &addr},
		CreatorMemberID:             "m1",
		OrganizerMemberIDs:          []domain.MemberID{"m1", "m2"},
		DraftVisibility:             porttriprepo.DraftVisibilityPublic,
		CreatedAt:                   now,
		UpdatedAt:                   now,
	})

	// Seed RSVPs: m1 YES, m2 NO.
	_ = rsvpsRepo.Upsert(ctx, portrsvprepo.RSVP{TripID: "tp1", MemberID: "m1", Status: portrsvprepo.StatusYes, UpdatedAt: now})
	_ = rsvpsRepo.Upsert(ctx, portrsvprepo.RSVP{TripID: "tp1", MemberID: "m2", Status: portrsvprepo.StatusNo, UpdatedAt: now})

	td, err := svc.GetTripDetails(ctx, "m1", "tp1")
	if err != nil {
		t.Fatalf("GetTripDetails: %v", err)
	}
	if !td.RSVPActionsEnabled {
		t.Fatalf("RSVPActionsEnabled should be true for published trips")
	}
	if td.RSVPSummary == nil || td.MyRSVP == nil {
		t.Fatalf("expected RSVP summary and myRSVP present for m1")
	}
	if td.MyRSVP.Response != domain.RSVPResponseYes {
		t.Fatalf("my=%+v", td.MyRSVP)
	}
	if td.RSVPSummary.AttendingRigs != 1 {
		t.Fatalf("AttendingRigs=%d want=1", td.RSVPSummary.AttendingRigs)
	}
}

func TestService_GetMyRSVPForTrip_HappyPathAndEdgeCases(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	membersRepo := memmemberrepo.NewRepo()
	tripsRepo := memtriprepo.NewRepo()
	rsvpsRepo := memrsvprepo.NewRepo()
	provisionMember(t, membersRepo, "m1")

	svc := trips.NewService(tripsRepo, membersRepo, rsvpsRepo)

	name := "Trip"
	now := time.Unix(1300, 0).UTC()
	_ = tripsRepo.Create(ctx, porttriprepo.Trip{ID: "draft", Status: porttriprepo.StatusDraft, Name: &name, DraftVisibility: porttriprepo.DraftVisibilityPrivate, CreatorMemberID: "m1", OrganizerMemberIDs: []domain.MemberID{"m1"}, CreatedAt: now, UpdatedAt: now})
	_ = tripsRepo.Create(ctx, porttriprepo.Trip{ID: "pub", Status: porttriprepo.StatusPublished, Name: &name, CreatedAt: now, UpdatedAt: now})

	// Draft => RSVP not available.
	if _, err := svc.GetMyRSVPForTrip(ctx, "m1", "draft"); err == nil {
		t.Fatalf("expected error for draft trip")
	}

	// Published but no RSVP => 404 RSVP_NOT_FOUND.
	_, err := svc.GetMyRSVPForTrip(ctx, "m1", "pub")
	if err == nil {
		t.Fatalf("expected error")
	}
	ae := (*trips.Error)(nil)
	if !errors.As(err, &ae) || ae.Status != 404 || ae.Code != "RSVP_NOT_FOUND" {
		t.Fatalf("err=%v", err)
	}

	_ = rsvpsRepo.Upsert(ctx, portrsvprepo.RSVP{TripID: "pub", MemberID: "m1", Status: portrsvprepo.StatusNo, UpdatedAt: now})
	my, err := svc.GetMyRSVPForTrip(ctx, "m1", "pub")
	if err != nil {
		t.Fatalf("GetMyRSVPForTrip: %v", err)
	}
	if my.Response != domain.RSVPResponseNo {
		t.Fatalf("my=%+v", my)
	}
}

func TestService_SetTripDraftVisibility_ValidatesAndAuthorizes(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	membersRepo := memmemberrepo.NewRepo()
	tripsRepo := memtriprepo.NewRepo()
	rsvpsRepo := memrsvprepo.NewRepo()
	provisionMember(t, membersRepo, "m1")
	provisionMember(t, membersRepo, "m2")

	svc := trips.NewService(tripsRepo, membersRepo, rsvpsRepo)

	name := "Draft"
	now := time.Unix(1400, 0).UTC()
	_ = tripsRepo.Create(ctx, porttriprepo.Trip{
		ID:                 "td1",
		Status:             porttriprepo.StatusDraft,
		Name:               &name,
		CreatorMemberID:    "m1",
		OrganizerMemberIDs: []domain.MemberID{"m1"},
		DraftVisibility:    porttriprepo.DraftVisibilityPrivate,
		CreatedAt:          now,
		UpdatedAt:          now,
	})

	// Invalid dv.
	_, err := svc.SetTripDraftVisibility(ctx, "m1", "td1", domain.DraftVisibility("BOGUS"))
	if err == nil {
		t.Fatalf("expected error")
	}
	ae := (*trips.Error)(nil)
	if !errors.As(err, &ae) || ae.Status != 422 {
		t.Fatalf("err=%v", err)
	}

	// Not creator => 404.
	_, err = svc.SetTripDraftVisibility(ctx, "m2", "td1", domain.DraftVisibilityPublic)
	if err == nil {
		t.Fatalf("expected error")
	}
	if !errors.As(err, &ae) || ae.Status != 404 {
		t.Fatalf("err=%v", err)
	}

	// Happy path: make public.
	td, err := svc.SetTripDraftVisibility(ctx, "m1", "td1", domain.DraftVisibilityPublic)
	if err != nil {
		t.Fatalf("SetTripDraftVisibility: %v", err)
	}
	if td.DraftVisibility == nil || *td.DraftVisibility != domain.DraftVisibilityPublic {
		t.Fatalf("draftVisibility=%v", td.DraftVisibility)
	}
}

func TestService_PublishTrip_HappyPathAndAnnouncementCopy(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	membersRepo := memmemberrepo.NewRepo()
	tripsRepo := memtriprepo.NewRepo()
	rsvpsRepo := memrsvprepo.NewRepo()
	provisionMember(t, membersRepo, "m1")

	svc := trips.NewService(tripsRepo, membersRepo, rsvpsRepo)

	name := "Snow Run"
	desc := "Fun trip"
	diff := "Easy"
	comms := "FRS"
	recs := "Spare tire"
	start := time.Date(2026, 4, 1, 0, 0, 0, 0, time.UTC)
	end := time.Date(2026, 4, 2, 0, 0, 0, 0, time.UTC)
	cap := 8
	addr := "Trailhead"
	now := time.Unix(1500, 0).UTC()
	_ = tripsRepo.Create(ctx, porttriprepo.Trip{
		ID:                          "tdp",
		Status:                      porttriprepo.StatusDraft,
		Name:                        &name,
		Description:                 &desc,
		DifficultyText:              &diff,
		CommsRequirementsText:       &comms,
		RecommendedRequirementsText: &recs,
		StartDate:                   &start,
		EndDate:                     &end,
		CapacityRigs:                &cap,
		MeetingLocation:             &domain.Location{Label: "Meet", Address: &addr},
		CreatorMemberID:             "m1",
		OrganizerMemberIDs:          []domain.MemberID{"m1"},
		DraftVisibility:             porttriprepo.DraftVisibilityPublic,
		CreatedAt:                   now,
		UpdatedAt:                   now,
	})

	td, copy, err := svc.PublishTrip(ctx, "m1", "tdp")
	if err != nil {
		t.Fatalf("PublishTrip: %v", err)
	}
	if td.Status != domain.TripStatusPublished {
		t.Fatalf("status=%s want=PUBLISHED", td.Status)
	}
	if copy == "" || !strings.Contains(copy, "Trip: Snow Run") {
		t.Fatalf("copy=%q", copy)
	}
	if td.AttendingRigs == nil || *td.AttendingRigs != 0 {
		t.Fatalf("attendingRigs=%v want=0", td.AttendingRigs)
	}

	// Idempotent publish on already-published trip should return copy.
	_, copy2, err := svc.PublishTrip(ctx, "m1", "tdp")
	if err != nil {
		t.Fatalf("PublishTrip2: %v", err)
	}
	if copy2 == "" {
		t.Fatalf("expected copy2")
	}
}

func TestService_UpdateTrip_MeetingLocationPatchAndArtifactReorder(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	membersRepo := memmemberrepo.NewRepo()
	tripsRepo := memtriprepo.NewRepo()
	rsvpsRepo := memrsvprepo.NewRepo()
	provisionMember(t, membersRepo, "m1")

	svc := trips.NewService(tripsRepo, membersRepo, rsvpsRepo)

	name := "Trip"
	now := time.Unix(1600, 0).UTC()
	addr := "Old"
	lat := 1.23
	lng := 4.56
	_ = tripsRepo.Create(ctx, porttriprepo.Trip{
		ID:                 "tp",
		Status:             porttriprepo.StatusPublished,
		Name:               &name,
		CreatorMemberID:    "m1",
		OrganizerMemberIDs: []domain.MemberID{"m1"},
		DraftVisibility:    porttriprepo.DraftVisibilityPublic,
		MeetingLocation:    &domain.Location{Label: "Meet", Address: &addr, Latitude: &lat, Longitude: &lng},
		Artifacts: []domain.TripArtifact{
			{ArtifactID: "a1", Type: domain.ArtifactTypeGPX, Title: "A1", URL: "https://example.com/a1"},
			{ArtifactID: "a2", Type: domain.ArtifactTypeDocument, Title: "A2", URL: "https://example.com/a2"},
		},
		CreatedAt: now,
		UpdatedAt: now,
	})

	// Reorder artifacts + clear coordinates + change address; label null should be ignored (kept).
	patch := &trips.LocationPatch{
		Label:            trips.Null[string](),
		Address:          trips.Some("New"),
		ClearCoordinates: true,
	}
	td, err := svc.UpdateTrip(ctx, "m1", "tp", trips.UpdateTripInput{
		MeetingLocation: trips.Some(patch),
		ArtifactIDs:     trips.Some([]string{"a2", "a1"}),
	})
	if err != nil {
		t.Fatalf("UpdateTrip: %v", err)
	}
	if td.MeetingLocation == nil || td.MeetingLocation.Address == nil || *td.MeetingLocation.Address != "New" {
		t.Fatalf("location=%+v", td.MeetingLocation)
	}
	if td.MeetingLocation.Label != "Meet" {
		t.Fatalf("label=%q want=%q", td.MeetingLocation.Label, "Meet")
	}
	if td.MeetingLocation.Latitude != nil || td.MeetingLocation.Longitude != nil {
		t.Fatalf("expected coordinates cleared: %+v", td.MeetingLocation)
	}
	if len(td.Artifacts) != 2 || td.Artifacts[0].ArtifactID != "a2" || td.Artifacts[1].ArtifactID != "a1" {
		t.Fatalf("artifacts=%v", td.Artifacts)
	}

	// Unknown artifact ID should return 422.
	_, err = svc.UpdateTrip(ctx, "m1", "tp", trips.UpdateTripInput{
		ArtifactIDs: trips.Some([]string{"nope"}),
	})
	if err == nil {
		t.Fatalf("expected error")
	}
	ae := (*trips.Error)(nil)
	if !errors.As(err, &ae) || ae.Status != 422 {
		t.Fatalf("err=%v", err)
	}

	// Invalid date range should return 422.
	start := time.Date(2026, 5, 2, 0, 0, 0, 0, time.UTC)
	end := time.Date(2026, 5, 1, 0, 0, 0, 0, time.UTC)
	_, err = svc.UpdateTrip(ctx, "m1", "tp", trips.UpdateTripInput{
		StartDate: trips.Some(start),
		EndDate:   trips.Some(end),
	})
	if err == nil {
		t.Fatalf("expected error")
	}
	if !errors.As(err, &ae) || ae.Status != 422 {
		t.Fatalf("err=%v", err)
	}
}

func containsAll(have []string, want []string) bool {
	set := make(map[string]struct{}, len(have))
	for _, v := range have {
		set[v] = struct{}{}
	}
	for _, w := range want {
		if _, ok := set[w]; !ok {
			return false
		}
	}
	return true
}
