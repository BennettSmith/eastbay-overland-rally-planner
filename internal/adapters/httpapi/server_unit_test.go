package httpapi

import (
	"context"
	"testing"
	"time"

	"github.com/oapi-codegen/nullable"
	openapi_types "github.com/oapi-codegen/runtime/types"

	"github.com/Overland-East-Bay/trip-planner-api/internal/adapters/httpapi/oas"
	memclock "github.com/Overland-East-Bay/trip-planner-api/internal/adapters/memory/clock"
	memidempotency "github.com/Overland-East-Bay/trip-planner-api/internal/adapters/memory/idempotency"
	memmemberrepo "github.com/Overland-East-Bay/trip-planner-api/internal/adapters/memory/memberrepo"
	memrsvprepo "github.com/Overland-East-Bay/trip-planner-api/internal/adapters/memory/rsvprepo"
	memtriprepo "github.com/Overland-East-Bay/trip-planner-api/internal/adapters/memory/triprepo"
	"github.com/Overland-East-Bay/trip-planner-api/internal/app/members"
	"github.com/Overland-East-Bay/trip-planner-api/internal/app/trips"
	"github.com/Overland-East-Bay/trip-planner-api/internal/domain"
)

func TestServer_CreateMyMember_MissingSubjectAndBodyValidation(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	clk := memclock.NewManualClock(time.Unix(1, 0).UTC())
	memberRepo := memmemberrepo.NewRepo()
	tripRepo := memtriprepo.NewRepo()
	rsvpRepo := memrsvprepo.NewRepo()
	idem := memidempotency.NewStore()
	s := NewServer(members.NewService(memberRepo, clk), trips.NewService(tripRepo, memberRepo, rsvpRepo), idem)

	// Missing subject -> 401.
	body := oas.CreateMyMemberJSONRequestBody(oas.CreateMemberRequest{
		DisplayName: "Alice",
		Email:       openapi_types.Email("alice@example.com"),
	})
	resp, err := s.CreateMyMember(ctx, oas.CreateMyMemberRequestObject{Body: &body})
	if err != nil {
		t.Fatalf("err=%v", err)
	}
	if _, ok := resp.(oas.CreateMyMember401JSONResponse); !ok {
		t.Fatalf("resp=%T want 401", resp)
	}

	// Missing body -> 422.
	ctx2 := WithSubject(ctx, "sub-1")
	resp, err = s.CreateMyMember(ctx2, oas.CreateMyMemberRequestObject{Body: nil})
	if err != nil {
		t.Fatalf("err2=%v", err)
	}
	if _, ok := resp.(oas.CreateMyMember422JSONResponse); !ok {
		t.Fatalf("resp2=%T want 422", resp)
	}

	// Invalid email -> 422.
	bodyBad := oas.CreateMyMemberJSONRequestBody(oas.CreateMemberRequest{
		DisplayName: "Alice",
		Email:       openapi_types.Email("not-an-email"),
	})
	resp, err = s.CreateMyMember(ctx2, oas.CreateMyMemberRequestObject{Body: &bodyBad})
	if err != nil {
		t.Fatalf("err3=%v", err)
	}
	if _, ok := resp.(oas.CreateMyMember422JSONResponse); !ok {
		t.Fatalf("resp3=%T want 422", resp)
	}
}

func TestVehicleProfileMapping_FromDomainAndFromOAS(t *testing.T) {
	t.Parallel()

	make := "Toyota"
	model := "4Runner"
	vp := domain.VehicleProfile{
		Make:  &make,
		Model: &model,
	}

	out := vehicleProfileFromDomain(vp)
	if out == nil {
		t.Fatalf("expected non-nil")
	}
	if !out.Make.IsSpecified() || out.Make.IsNull() {
		t.Fatalf("make not specified")
	}
	if got, _ := out.Make.Get(); got != "Toyota" {
		t.Fatalf("make=%q", got)
	}

	var o oas.VehicleProfile
	o.Make = nullable.NewNullableWithValue("Jeep")
	o.Model = nullable.NewNullableWithValue("Wrangler")
	patch := vehicleProfilePatchFromOAS(o)
	if patch == nil || !patch.Make.IsSpecified() || patch.Make.IsNull() || patch.Make.Value() != "Jeep" {
		t.Fatalf("patch=%+v", patch)
	}
}

func TestOptionalFloatFromNullableTrips(t *testing.T) {
	t.Parallel()

	var u nullable.Nullable[float64]
	if got := optionalFloatFromNullableTrips(u); got.IsSpecified() || got.IsNull() {
		t.Fatalf("unspecified=%+v", got)
	}

	var n nullable.Nullable[float64]
	n.SetNull()
	if got := optionalFloatFromNullableTrips(n); !got.IsSpecified() || !got.IsNull() {
		t.Fatalf("null=%+v", got)
	}

	var v nullable.Nullable[float64]
	v.Set(1.23)
	got := optionalFloatFromNullableTrips(v)
	if !got.IsSpecified() || got.IsNull() || got.Value() != 1.23 {
		t.Fatalf("value=%+v", got)
	}
}
