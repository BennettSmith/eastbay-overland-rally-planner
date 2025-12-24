package httpapi

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	memclock "eastbay-overland-rally-planner/internal/adapters/memory/clock"
	memidempotency "eastbay-overland-rally-planner/internal/adapters/memory/idempotency"
	memmemberrepo "eastbay-overland-rally-planner/internal/adapters/memory/memberrepo"
	memtriprepo "eastbay-overland-rally-planner/internal/adapters/memory/triprepo"
	"eastbay-overland-rally-planner/internal/adapters/httpapi/oas"
	"eastbay-overland-rally-planner/internal/app/members"
	"eastbay-overland-rally-planner/internal/app/trips"
	"eastbay-overland-rally-planner/internal/domain"
	"eastbay-overland-rally-planner/internal/platform/auth/jwks_testutil"
	"eastbay-overland-rally-planner/internal/platform/auth/jwtverifier"
	"eastbay-overland-rally-planner/internal/platform/config"
	portmemberrepo "eastbay-overland-rally-planner/internal/ports/out/memberrepo"
	porttriprepo "eastbay-overland-rally-planner/internal/ports/out/triprepo"
)

type fixedClockTrips struct{ t time.Time }

func (c fixedClockTrips) Now() time.Time { return c.t }

func newTestTripRouter(t *testing.T) (http.Handler, func(now time.Time, kid string) string, *memtriprepo.Repo, *memmemberrepo.Repo) {
	t.Helper()

	kp, err := jwks_testutil.GenerateRSAKeypair("kid-1")
	if err != nil {
		t.Fatalf("GenerateRSAKeypair: %v", err)
	}
	jwksSrv, setKeys := jwks_testutil.NewRotatingJWKSServer()
	t.Cleanup(jwksSrv.Close)
	setKeys([]jwks_testutil.Keypair{kp})

	jwtCfg := config.JWTConfig{
		Issuer:                  "test-iss",
		Audience:                "test-aud",
		JWKSURL:                 jwksSrv.URL,
		ClockSkew:               0,
		JWKSRefreshInterval:     10 * time.Minute,
		JWKSMinRefreshInterval: time.Second,
		HTTPTimeout:            2 * time.Second,
	}
	v := jwtverifier.NewWithOptions(jwtCfg, nil, fixedClockTrips{t: time.Unix(1700000000, 0)})

	clk := memclock.NewManualClock(time.Unix(100, 0).UTC())
	memberRepo := memmemberrepo.NewRepo()
	tripRepo := memtriprepo.NewRepo()
	idem := memidempotency.NewStore()
	memberSvc := members.NewService(memberRepo, clk)
	tripSvc := trips.NewService(tripRepo, memberRepo)

	api := NewServer(memberSvc, tripSvc, idem)
	h := NewRouterWithOptions(api, RouterOptions{AuthMiddleware: NewAuthMiddleware(v)})

	mint := func(now time.Time, kid string) string {
		jwt, err := jwks_testutil.MintRS256JWT(
			jwks_testutil.Keypair{Kid: kid, Private: kp.Private},
			jwtCfg.Issuer,
			jwtCfg.Audience,
			"sub-1",
			now,
			10*time.Minute,
			nil,
		)
		if err != nil {
			t.Fatalf("MintRS256JWT: %v", err)
		}
		return jwt
	}

	return h, mint, tripRepo, memberRepo
}

func provisionCaller(t *testing.T, h http.Handler, authz string) domain.MemberID {
	t.Helper()

	req := httptest.NewRequest(http.MethodPost, "/members", bytes.NewBufferString(`{"displayName":"Alice","email":"alice@example.com"}`))
	req.Header.Set("Authorization", authz)
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusCreated {
		t.Fatalf("provision status=%d body=%s", rec.Code, rec.Body.String())
	}

	var payload struct {
		Member oas.MemberProfile `json:"member"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode provision: %v", err)
	}
	return domain.MemberID(payload.Member.MemberId)
}

func TestTrips_ListVisibleTripsForMember_FiltersAndSorts(t *testing.T) {
	t.Parallel()

	h, mint, tripRepo, _ := newTestTripRouter(t)
	authz := "Bearer " + mint(time.Unix(1700000000, 0), "kid-1")
	_ = provisionCaller(t, h, authz)

	start1 := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	start2 := time.Date(2026, 1, 2, 0, 0, 0, 0, time.UTC)

	// Undated published sorts after dated, by CreatedAt.
	_ = tripRepo.Create(context.Background(), porttriprepo.Trip{
		ID:        "t3",
		Status:    porttriprepo.StatusPublished,
		CreatedAt: time.Unix(20, 0).UTC(),
	})
	_ = tripRepo.Create(context.Background(), porttriprepo.Trip{
		ID:        "t2",
		Status:    porttriprepo.StatusCanceled,
		StartDate: &start2,
		CreatedAt: time.Unix(30, 0).UTC(),
	})
	_ = tripRepo.Create(context.Background(), porttriprepo.Trip{
		ID:        "t1",
		Status:    porttriprepo.StatusPublished,
		StartDate: &start1,
		CreatedAt: time.Unix(10, 0).UTC(),
	})
	_ = tripRepo.Create(context.Background(), porttriprepo.Trip{
		ID:        "t4",
		Status:    porttriprepo.StatusDraft,
		CreatedAt: time.Unix(40, 0).UTC(),
	})

	req := httptest.NewRequest(http.MethodGet, "/trips", nil)
	req.Header.Set("Authorization", authz)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}

	var resp struct {
		Trips []oas.TripSummary `json:"trips"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(resp.Trips) != 3 {
		t.Fatalf("len=%d want=3", len(resp.Trips))
	}
	if resp.Trips[0].TripId != "t1" || resp.Trips[1].TripId != "t2" || resp.Trips[2].TripId != "t3" {
		t.Fatalf("order=%v want=[t1 t2 t3]", []string{resp.Trips[0].TripId, resp.Trips[1].TripId, resp.Trips[2].TripId})
	}
	// UC-01: for non-draft, draftVisibility omitted.
	if resp.Trips[0].DraftVisibility != nil || resp.Trips[1].DraftVisibility != nil || resp.Trips[2].DraftVisibility != nil {
		t.Fatalf("draftVisibility should be omitted for non-drafts")
	}
}

func TestTrips_ListMyDraftTrips_VisibilityAndDraftVisibilityField(t *testing.T) {
	t.Parallel()

	h, mint, tripRepo, memberRepo := newTestTripRouter(t)
	authz := "Bearer " + mint(time.Unix(1700000000, 0), "kid-1")
	callerID := provisionCaller(t, h, authz)

	// Extra members (for organizer ID references).
	_ = memberRepo.Create(context.Background(), portmemberrepo.Member{
		ID:          "m2",
		Subject:     "sub-2",
		DisplayName: "Bob",
		Email:       "bob@example.com",
		IsActive:    true,
		CreatedAt:   time.Unix(2, 0).UTC(),
		UpdatedAt:   time.Unix(2, 0).UTC(),
	})

	// Visible: PUBLIC + caller is organizer
	_ = tripRepo.Create(context.Background(), porttriprepo.Trip{
		ID:                "t1",
		Status:            porttriprepo.StatusDraft,
		DraftVisibility:   porttriprepo.DraftVisibilityPublic,
		OrganizerMemberIDs: []domain.MemberID{callerID, "m2"},
		CreatedAt:         time.Unix(10, 0).UTC(),
	})
	// Not visible: PUBLIC + caller not organizer
	_ = tripRepo.Create(context.Background(), porttriprepo.Trip{
		ID:                "t2",
		Status:            porttriprepo.StatusDraft,
		DraftVisibility:   porttriprepo.DraftVisibilityPublic,
		OrganizerMemberIDs: []domain.MemberID{"m2"},
		CreatedAt:         time.Unix(20, 0).UTC(),
	})
	// Visible: PRIVATE + caller is creator
	_ = tripRepo.Create(context.Background(), porttriprepo.Trip{
		ID:              "t3",
		Status:          porttriprepo.StatusDraft,
		DraftVisibility: porttriprepo.DraftVisibilityPrivate,
		CreatorMemberID: callerID,
		CreatedAt:       time.Unix(30, 0).UTC(),
	})
	// Not visible: PRIVATE + caller not creator
	_ = tripRepo.Create(context.Background(), porttriprepo.Trip{
		ID:              "t4",
		Status:          porttriprepo.StatusDraft,
		DraftVisibility: porttriprepo.DraftVisibilityPrivate,
		CreatorMemberID: "m2",
		CreatedAt:       time.Unix(40, 0).UTC(),
	})

	req := httptest.NewRequest(http.MethodGet, "/trips/drafts", nil)
	req.Header.Set("Authorization", authz)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}

	var resp struct {
		Trips []oas.TripSummary `json:"trips"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(resp.Trips) != 2 {
		t.Fatalf("len=%d want=2", len(resp.Trips))
	}
	if resp.Trips[0].TripId != "t1" || resp.Trips[1].TripId != "t3" {
		t.Fatalf("order=%v want=[t1 t3]", []string{resp.Trips[0].TripId, resp.Trips[1].TripId})
	}
	// UC-01: for drafts, draftVisibility included.
	if resp.Trips[0].DraftVisibility == nil || resp.Trips[1].DraftVisibility == nil {
		t.Fatalf("draftVisibility should be present for drafts")
	}
}

func TestTrips_GetTripDetails_VisibilityRulesAndResponseShape(t *testing.T) {
	t.Parallel()

	h, mint, tripRepo, memberRepo := newTestTripRouter(t)
	authz := "Bearer " + mint(time.Unix(1700000000, 0), "kid-1")
	callerID := provisionCaller(t, h, authz)

	// Add another organizer member so expansion works.
	_ = memberRepo.Create(context.Background(), portmemberrepo.Member{
		ID:          "m2",
		Subject:     "sub-2",
		DisplayName: "Bob",
		Email:       "bob@example.com",
		IsActive:    true,
		CreatedAt:   time.Unix(2, 0).UTC(),
		UpdatedAt:   time.Unix(2, 0).UTC(),
	})

	name := "Snow Run"
	desc := "Fun winter trip"
	cap := 10
	addr := "Somewhere"
	lat := 37.0
	lng := -122.0
	start := time.Date(2026, 2, 1, 0, 0, 0, 0, time.UTC)
	end := time.Date(2026, 2, 2, 0, 0, 0, 0, time.UTC)

	_ = tripRepo.Create(context.Background(), porttriprepo.Trip{
		ID:                 "tp",
		Status:             porttriprepo.StatusPublished,
		Name:               &name,
		Description:        &desc,
		StartDate:           &start,
		EndDate:             &end,
		CapacityRigs:        &cap,
		CreatorMemberID:     callerID,
		OrganizerMemberIDs:  []domain.MemberID{callerID, "m2"},
		MeetingLocation:     &domain.Location{Label: "Meet", Address: &addr, Latitude: &lat, Longitude: &lng},
		Artifacts:           []domain.TripArtifact{{ArtifactID: "a1", Type: domain.ArtifactTypeGPX, Title: "Route", URL: "https://example.com/route.gpx"}},
		CreatedAt:          time.Unix(10, 0).UTC(),
	})

	// Non-visible draft should 404.
	_ = tripRepo.Create(context.Background(), porttriprepo.Trip{
		ID:              "td-private",
		Status:          porttriprepo.StatusDraft,
		DraftVisibility: porttriprepo.DraftVisibilityPrivate,
		CreatorMemberID: "m2",
		OrganizerMemberIDs: []domain.MemberID{"m2"},
		CreatedAt:       time.Unix(20, 0).UTC(),
	})

	req404 := httptest.NewRequest(http.MethodGet, "/trips/td-private", nil)
	req404.Header.Set("Authorization", authz)
	rec404 := httptest.NewRecorder()
	h.ServeHTTP(rec404, req404)
	if rec404.Code != http.StatusNotFound {
		t.Fatalf("status=%d body=%s", rec404.Code, rec404.Body.String())
	}

	req := httptest.NewRequest(http.MethodGet, "/trips/tp", nil)
	req.Header.Set("Authorization", authz)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	var resp struct {
		Trip oas.TripDetails `json:"trip"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if resp.Trip.TripId != "tp" || resp.Trip.Status != "PUBLISHED" {
		t.Fatalf("tripId/status=%s/%s", resp.Trip.TripId, resp.Trip.Status)
	}
	if !resp.Trip.RsvpActionsEnabled {
		t.Fatalf("rsvpActionsEnabled should be true for published trips")
	}
	if len(resp.Trip.Organizers) != 2 || len(resp.Trip.Artifacts) != 1 {
		t.Fatalf("organizers=%d artifacts=%d", len(resp.Trip.Organizers), len(resp.Trip.Artifacts))
	}
}


