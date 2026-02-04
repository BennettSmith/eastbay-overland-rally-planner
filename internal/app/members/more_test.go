package members

import (
	"context"
	"errors"
	"testing"
	"time"

	memclock "github.com/Overland-East-Bay/trip-planner-api/internal/adapters/memory/clock"
	memmemberrepo "github.com/Overland-East-Bay/trip-planner-api/internal/adapters/memory/memberrepo"
	"github.com/Overland-East-Bay/trip-planner-api/internal/domain"
	portmemberrepo "github.com/Overland-East-Bay/trip-planner-api/internal/ports/out/memberrepo"
)

func TestError_ErrorStringAndWithDetailsCopies(t *testing.T) {
	t.Parallel()

	if (*Error)(nil).Error() != "<nil>" {
		t.Fatalf("nil error string mismatch")
	}

	e := &Error{Status: 409, Code: "CODE", Message: "msg"}
	if got := e.Error(); got != "CODE: msg" {
		t.Fatalf("Error()=%q", got)
	}

	e2 := &Error{Status: 500, Message: "boom"}
	if got := e2.Error(); got != "app error (status=500): boom" {
		t.Fatalf("Error()=%q", got)
	}

	d := map[string]any{"a": 1}
	e3 := e.WithDetails(d)
	if e3 == nil || e3.Details["a"] != 1 {
		t.Fatalf("WithDetails=%+v", e3)
	}
	// Ensure copy (not shared).
	d["a"] = 2
	if e3.Details["a"] != 1 {
		t.Fatalf("expected details map copied")
	}
}

func TestOptional_TriStateSemantics(t *testing.T) {
	t.Parallel()

	u := Unspecified[string]()
	if u.IsSpecified() || u.IsNull() || u.Value() != "" {
		t.Fatalf("unspecified=%+v", u)
	}

	n := Null[string]()
	if !n.IsSpecified() || !n.IsNull() {
		t.Fatalf("null=%+v", n)
	}

	s := Some("x")
	if !s.IsSpecified() || s.IsNull() || s.Value() != "x" {
		t.Fatalf("some=%+v", s)
	}
}

func TestService_ListMembers_IncludeInactiveAndIncludesInactiveCaller(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	repo := memmemberrepo.NewRepo()
	clk := memclock.NewManualClock(time.Unix(100, 0).UTC())
	svc := NewService(repo, clk)

	now := clk.Now()
	_ = repo.Create(ctx, portmemberrepo.Member{ID: "m1", Subject: "sub-1", DisplayName: "Zoe", Email: "zoe@example.com", IsActive: false, CreatedAt: now, UpdatedAt: now})
	_ = repo.Create(ctx, portmemberrepo.Member{ID: "m2", Subject: "sub-2", DisplayName: "Alice", Email: "alice@example.com", IsActive: true, CreatedAt: now, UpdatedAt: now})
	_ = repo.Create(ctx, portmemberrepo.Member{ID: "m3", Subject: "sub-3", DisplayName: "Bob", Email: "bob@example.com", IsActive: false, CreatedAt: now, UpdatedAt: now})

	// includeInactive=false should still include inactive caller ("sub-1"), but not other inactive members.
	got, err := svc.ListMembers(ctx, domain.SubjectID("sub-1"), false)
	if err != nil {
		t.Fatalf("ListMembers: %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("len=%d want=2", len(got))
	}
	// Sorted case-insensitively by displayName.
	if got[0].ID != "m2" || got[1].ID != "m1" {
		t.Fatalf("order=%v", []domain.MemberID{got[0].ID, got[1].ID})
	}

	// includeInactive=true returns all members (still sorted).
	got2, err := svc.ListMembers(ctx, domain.SubjectID("sub-1"), true)
	if err != nil {
		t.Fatalf("ListMembers(includeInactive=true): %v", err)
	}
	if len(got2) != 3 {
		t.Fatalf("len=%d want=3", len(got2))
	}
}

func TestService_UpdateMyMemberProfile_EmailUniquenessAndVehicleProfilePatch(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	repo := memmemberrepo.NewRepo()
	clk := memclock.NewManualClock(time.Unix(100, 0).UTC())
	svc := NewService(repo, clk)

	// Create two members.
	_, err := svc.CreateMyMember(ctx, "sub-1", CreateMyMemberInput{DisplayName: "Alice", Email: "alice@example.com"})
	if err != nil {
		t.Fatalf("CreateMyMember: %v", err)
	}
	_, err = svc.CreateMyMember(ctx, "sub-2", CreateMyMemberInput{DisplayName: "Bob", Email: "bob@example.com"})
	if err != nil {
		t.Fatalf("CreateMyMember2: %v", err)
	}

	// Attempt to change sub-1's email to bob@example.com should conflict.
	_, err = svc.UpdateMyMemberProfile(ctx, "sub-1", UpdateMyMemberProfileInput{
		Email: Some("bob@example.com"),
	})
	if err == nil {
		t.Fatalf("expected error")
	}
	ae := (*Error)(nil)
	if !errors.As(err, &ae) || ae.Status != 409 || ae.Code != "EMAIL_ALREADY_IN_USE" {
		t.Fatalf("err=%v", err)
	}

	// Apply a vehicle profile patch to nil existing profile.
	updated, err := svc.UpdateMyMemberProfile(ctx, "sub-1", UpdateMyMemberProfileInput{
		VehicleProfile: Some(VehicleProfilePatch{
			Make:  Some("Toyota"),
			Model: Some("4Runner"),
		}),
	})
	if err != nil {
		t.Fatalf("UpdateMyMemberProfile(vehicle): %v", err)
	}
	if updated.VehicleProfile == nil || updated.VehicleProfile.Make == nil || *updated.VehicleProfile.Make != "Toyota" {
		t.Fatalf("vehicleProfile=%+v", updated.VehicleProfile)
	}

	// Clear a field via null.
	updated2, err := svc.UpdateMyMemberProfile(ctx, "sub-1", UpdateMyMemberProfileInput{
		VehicleProfile: Some(VehicleProfilePatch{
			Make: Null[string](),
		}),
	})
	if err != nil {
		t.Fatalf("UpdateMyMemberProfile(vehicle clear): %v", err)
	}
	if updated2.VehicleProfile == nil || updated2.VehicleProfile.Make != nil {
		t.Fatalf("expected Make cleared: %+v", updated2.VehicleProfile)
	}
}

func TestService_SearchMembers_TrimsAndRespectsLimit(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	repo := memmemberrepo.NewRepo()
	clk := memclock.NewManualClock(time.Unix(100, 0).UTC())
	svc := NewService(repo, clk)
	svc.SearchLimit = 1

	now := clk.Now()
	_ = repo.Create(ctx, portmemberrepo.Member{ID: "m1", Subject: "sub-1", DisplayName: "Alice Smith", Email: "a1@example.com", IsActive: true, CreatedAt: now, UpdatedAt: now})
	_ = repo.Create(ctx, portmemberrepo.Member{ID: "m2", Subject: "sub-2", DisplayName: "Alice Jones", Email: "a2@example.com", IsActive: true, CreatedAt: now, UpdatedAt: now})
	_ = repo.Create(ctx, portmemberrepo.Member{ID: "m3", Subject: "sub-3", DisplayName: "Bob", Email: "b@example.com", IsActive: true, CreatedAt: now, UpdatedAt: now})

	got, err := svc.SearchMembers(ctx, "  ali  ")
	if err != nil {
		t.Fatalf("SearchMembers: %v", err)
	}
	if len(got) != 1 {
		t.Fatalf("len=%d want=1", len(got))
	}
	if got[0].ID != "m1" && got[0].ID != "m2" {
		t.Fatalf("unexpected member: %+v", got[0])
	}
}

func TestService_AnonymizeAndDeactivateMyMember_ScrubsFields(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	repo := memmemberrepo.NewRepo()
	clk := memclock.NewManualClock(time.Unix(100, 0).UTC())
	svc := NewService(repo, clk)

	// Create member with extra fields.
	gae := "alias@example.com"
	_, err := svc.CreateMyMember(ctx, "sub-1", CreateMyMemberInput{
		DisplayName:     "Alice",
		Email:           "alice@example.com",
		GroupAliasEmail: &gae,
		VehicleProfile: &VehicleProfilePatch{
			Make: Some("Toyota"),
		},
	})
	if err != nil {
		t.Fatalf("CreateMyMember: %v", err)
	}

	clk.Add(10 * time.Second)
	if err := svc.AnonymizeAndDeactivateMyMember(ctx, "sub-1"); err != nil {
		t.Fatalf("AnonymizeAndDeactivateMyMember: %v", err)
	}

	// Fetch directly from repo to validate persistence changes.
	m, err := repo.GetBySubject(ctx, "sub-1")
	if err != nil {
		t.Fatalf("GetBySubject: %v", err)
	}
	if m.IsActive {
		t.Fatalf("expected inactive")
	}
	if m.DisplayName != "Deleted member" {
		t.Fatalf("displayName=%q", m.DisplayName)
	}
	if m.GroupAliasEmail != nil || m.VehicleProfile != nil {
		t.Fatalf("expected groupAliasEmail and vehicleProfile cleared")
	}
	if m.Email == "alice@example.com" || m.Email == "" {
		t.Fatalf("expected email placeholder, got=%q", m.Email)
	}

	// App-layer profile lookup should now behave as not provisioned.
	_, err = svc.GetMyMemberProfile(ctx, "sub-1")
	if err == nil {
		t.Fatalf("expected error")
	}
	ae := (*Error)(nil)
	if !errors.As(err, &ae) || ae.Status != 404 || ae.Code != "MEMBER_NOT_PROVISIONED" {
		t.Fatalf("err=%v", err)
	}
}
