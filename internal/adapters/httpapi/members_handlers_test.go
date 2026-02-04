package httpapi

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/Overland-East-Bay/trip-planner-api/internal/adapters/httpapi/oas"
	memclock "github.com/Overland-East-Bay/trip-planner-api/internal/adapters/memory/clock"
	memidempotency "github.com/Overland-East-Bay/trip-planner-api/internal/adapters/memory/idempotency"
	memmemberrepo "github.com/Overland-East-Bay/trip-planner-api/internal/adapters/memory/memberrepo"
	memrsvprepo "github.com/Overland-East-Bay/trip-planner-api/internal/adapters/memory/rsvprepo"
	memtriprepo "github.com/Overland-East-Bay/trip-planner-api/internal/adapters/memory/triprepo"
	"github.com/Overland-East-Bay/trip-planner-api/internal/app/members"
	"github.com/Overland-East-Bay/trip-planner-api/internal/app/trips"
	"github.com/Overland-East-Bay/trip-planner-api/internal/platform/auth/jwks_testutil"
	"github.com/Overland-East-Bay/trip-planner-api/internal/platform/auth/jwtverifier"
	"github.com/Overland-East-Bay/trip-planner-api/internal/platform/config"
	portmemberrepo "github.com/Overland-East-Bay/trip-planner-api/internal/ports/out/memberrepo"
)

type fixedClockMembers struct{ t time.Time }

func (c fixedClockMembers) Now() time.Time { return c.t }

func newTestMemberRouter(t *testing.T) (http.Handler, func(now time.Time, kid string) string) {
	t.Helper()

	kp, err := jwks_testutil.GenerateRSAKeypair("kid-1")
	if err != nil {
		t.Fatalf("GenerateRSAKeypair: %v", err)
	}
	jwksSrv, setKeys := jwks_testutil.NewRotatingJWKSServer()
	t.Cleanup(jwksSrv.Close)
	setKeys([]jwks_testutil.Keypair{kp})

	jwtCfg := config.JWTConfig{
		Issuer:                 "test-iss",
		Audience:               "test-aud",
		JWKSURL:                jwksSrv.URL,
		ClockSkew:              0,
		JWKSRefreshInterval:    10 * time.Minute,
		JWKSMinRefreshInterval: time.Second,
		HTTPTimeout:            2 * time.Second,
	}
	v := jwtverifier.NewWithOptions(jwtCfg, nil, fixedClockMembers{t: time.Unix(1700000000, 0)})

	clk := memclock.NewManualClock(time.Unix(100, 0).UTC())
	repo := memmemberrepo.NewRepo()
	idem := memidempotency.NewStore()
	memberSvc := members.NewService(repo, clk)

	tripRepo := memtriprepo.NewRepo()
	rsvpRepo := memrsvprepo.NewRepo()
	tripSvc := trips.NewService(tripRepo, repo, rsvpRepo)
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

	return h, mint
}

func TestMembers_GetMe_NotProvisioned_404(t *testing.T) {
	t.Parallel()

	h, mint := newTestMemberRouter(t)

	req := httptest.NewRequest(http.MethodGet, "/members/me", nil)
	req.Header.Set("Authorization", "Bearer "+mint(time.Unix(1700000000, 0), "kid-1"))
	rec := httptest.NewRecorder()

	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	var er oas.ErrorResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &er); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if er.Error.Code != "MEMBER_NOT_PROVISIONED" {
		t.Fatalf("code=%q", er.Error.Code)
	}
}

func TestMembers_CreateThenGetMe_200(t *testing.T) {
	t.Parallel()

	h, mint := newTestMemberRouter(t)

	createBody := `{"displayName":"Alice Smith","email":"alice@example.com"}`
	req := httptest.NewRequest(http.MethodPost, "/members", bytes.NewBufferString(createBody))
	req.Header.Set("Authorization", "Bearer "+mint(time.Unix(1700000000, 0), "kid-1"))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusCreated {
		t.Fatalf("create status=%d body=%s", rec.Code, rec.Body.String())
	}

	req2 := httptest.NewRequest(http.MethodGet, "/members/me", nil)
	req2.Header.Set("Authorization", "Bearer "+mint(time.Unix(1700000000, 0), "kid-1"))
	rec2 := httptest.NewRecorder()
	h.ServeHTTP(rec2, req2)
	if rec2.Code != http.StatusOK {
		t.Fatalf("get status=%d body=%s", rec2.Code, rec2.Body.String())
	}
}

func TestMembers_UpdateMe_IdempotentReplayAndConflictOnReuse(t *testing.T) {
	t.Parallel()

	h, mint := newTestMemberRouter(t)
	authz := "Bearer " + mint(time.Unix(1700000000, 0), "kid-1")

	// Provision first.
	reqCreate := httptest.NewRequest(http.MethodPost, "/members", bytes.NewBufferString(`{"displayName":"Alice","email":"alice@example.com"}`))
	reqCreate.Header.Set("Authorization", authz)
	reqCreate.Header.Set("Content-Type", "application/json")
	recCreate := httptest.NewRecorder()
	h.ServeHTTP(recCreate, reqCreate)
	if recCreate.Code != http.StatusCreated {
		t.Fatalf("create status=%d body=%s", recCreate.Code, recCreate.Body.String())
	}

	idemKey := "idem-12345678"
	body1 := `{"displayName":"  Alice   Smith "}`
	req1 := httptest.NewRequest(http.MethodPatch, "/members/me", bytes.NewBufferString(body1))
	req1.Header.Set("Authorization", authz)
	req1.Header.Set("Content-Type", "application/json")
	req1.Header.Set("Idempotency-Key", idemKey)
	rec1 := httptest.NewRecorder()
	h.ServeHTTP(rec1, req1)
	if rec1.Code != http.StatusOK {
		t.Fatalf("patch1 status=%d body=%s", rec1.Code, rec1.Body.String())
	}

	// Same key + same semantic payload (after normalization) should replay.
	body2 := `{"displayName":"Alice   Smith"}`
	req2 := httptest.NewRequest(http.MethodPatch, "/members/me", bytes.NewBufferString(body2))
	req2.Header.Set("Authorization", authz)
	req2.Header.Set("Content-Type", "application/json")
	req2.Header.Set("Idempotency-Key", idemKey)
	rec2 := httptest.NewRecorder()
	h.ServeHTTP(rec2, req2)
	if rec2.Code != http.StatusOK {
		t.Fatalf("patch2 status=%d body=%s", rec2.Code, rec2.Body.String())
	}

	// Same key + different payload should be 409.
	body3 := `{"displayName":"Bob"}` // different
	req3 := httptest.NewRequest(http.MethodPatch, "/members/me", bytes.NewBufferString(body3))
	req3.Header.Set("Authorization", authz)
	req3.Header.Set("Content-Type", "application/json")
	req3.Header.Set("Idempotency-Key", idemKey)
	rec3 := httptest.NewRecorder()
	h.ServeHTTP(rec3, req3)
	if rec3.Code != http.StatusConflict {
		t.Fatalf("patch3 status=%d body=%s", rec3.Code, rec3.Body.String())
	}
}

func TestMembers_DeleteMe_RequiresConfirm_409(t *testing.T) {
	t.Parallel()

	h, mint := newTestMemberRouter(t)
	authz := "Bearer " + mint(time.Unix(1700000000, 0), "kid-1")

	// Provision first.
	reqCreate := httptest.NewRequest(http.MethodPost, "/members", bytes.NewBufferString(`{"displayName":"Alice","email":"alice@example.com"}`))
	reqCreate.Header.Set("Authorization", authz)
	reqCreate.Header.Set("Content-Type", "application/json")
	recCreate := httptest.NewRecorder()
	h.ServeHTTP(recCreate, reqCreate)
	if recCreate.Code != http.StatusCreated {
		t.Fatalf("create status=%d body=%s", recCreate.Code, recCreate.Body.String())
	}

	req := httptest.NewRequest(http.MethodDelete, "/members/me", bytes.NewBufferString(`{"confirm":false}`))
	req.Header.Set("Authorization", authz)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Idempotency-Key", "idem-del-1")
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusConflict {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
}

func TestMembers_DeleteMe_200_ThenGetMe_404(t *testing.T) {
	t.Parallel()

	h, mint := newTestMemberRouter(t)
	authz := "Bearer " + mint(time.Unix(1700000000, 0), "kid-1")

	// Provision first.
	reqCreate := httptest.NewRequest(http.MethodPost, "/members", bytes.NewBufferString(`{"displayName":"Alice","email":"alice@example.com"}`))
	reqCreate.Header.Set("Authorization", authz)
	reqCreate.Header.Set("Content-Type", "application/json")
	recCreate := httptest.NewRecorder()
	h.ServeHTTP(recCreate, reqCreate)
	if recCreate.Code != http.StatusCreated {
		t.Fatalf("create status=%d body=%s", recCreate.Code, recCreate.Body.String())
	}

	req := httptest.NewRequest(http.MethodDelete, "/members/me", bytes.NewBufferString(`{"confirm":true,"reason":"testing"}`))
	req.Header.Set("Authorization", authz)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Idempotency-Key", "idem-del-2")
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("delete status=%d body=%s", rec.Code, rec.Body.String())
	}

	req2 := httptest.NewRequest(http.MethodGet, "/members/me", nil)
	req2.Header.Set("Authorization", authz)
	rec2 := httptest.NewRecorder()
	h.ServeHTTP(rec2, req2)
	if rec2.Code != http.StatusNotFound {
		t.Fatalf("get status=%d body=%s", rec2.Code, rec2.Body.String())
	}
}

func TestMembers_ListAndSearchMembers_ProvisioningAndIncludeInactive(t *testing.T) {
	t.Parallel()

	h, mint, _, memberRepo := newTestTripRouter(t)
	authz := "Bearer " + mint(time.Unix(1700000000, 0), "kid-1", "sub-1")

	// Must be provisioned to access the directory.
	req0 := httptest.NewRequest(http.MethodGet, "/members", nil)
	req0.Header.Set("Authorization", authz)
	rec0 := httptest.NewRecorder()
	h.ServeHTTP(rec0, req0)
	if rec0.Code != http.StatusUnauthorized {
		t.Fatalf("status=%d body=%s", rec0.Code, rec0.Body.String())
	}

	_ = provisionCaller(t, h, authz, "alice1@example.com")

	// Add another inactive member directly.
	now := time.Unix(200, 0).UTC()
	_ = memberRepo.Create(context.Background(), portmemberrepo.Member{
		ID:          "m2",
		Subject:     "sub-2",
		DisplayName: "Bob",
		Email:       "bob@example.com",
		IsActive:    false,
		CreatedAt:   now,
		UpdatedAt:   now,
	})

	req1 := httptest.NewRequest(http.MethodGet, "/members", nil)
	req1.Header.Set("Authorization", authz)
	rec1 := httptest.NewRecorder()
	h.ServeHTTP(rec1, req1)
	if rec1.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rec1.Code, rec1.Body.String())
	}
	var list1 struct {
		Members []oas.MemberDirectoryEntry `json:"members"`
	}
	if err := json.Unmarshal(rec1.Body.Bytes(), &list1); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(list1.Members) != 1 {
		t.Fatalf("len=%d want=1", len(list1.Members))
	}

	req2 := httptest.NewRequest(http.MethodGet, "/members?includeInactive=true", nil)
	req2.Header.Set("Authorization", authz)
	rec2 := httptest.NewRecorder()
	h.ServeHTTP(rec2, req2)
	if rec2.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rec2.Code, rec2.Body.String())
	}
	var list2 struct {
		Members []oas.MemberDirectoryEntry `json:"members"`
	}
	if err := json.Unmarshal(rec2.Body.Bytes(), &list2); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(list2.Members) != 2 {
		t.Fatalf("len=%d want=2", len(list2.Members))
	}

	// Search should validate min length (422), then succeed.
	reqBad := httptest.NewRequest(http.MethodGet, "/members/search?q=ab", nil)
	reqBad.Header.Set("Authorization", authz)
	recBad := httptest.NewRecorder()
	h.ServeHTTP(recBad, reqBad)
	if recBad.Code != http.StatusUnprocessableEntity {
		t.Fatalf("status=%d body=%s", recBad.Code, recBad.Body.String())
	}

	// Add an active member that matches.
	_ = memberRepo.Create(context.Background(), portmemberrepo.Member{
		ID:          "m3",
		Subject:     "sub-3",
		DisplayName: "Alice Smith",
		Email:       "alice2@example.com",
		IsActive:    true,
		CreatedAt:   now,
		UpdatedAt:   now,
	})
	reqOK := httptest.NewRequest(http.MethodGet, "/members/search?q=ali", nil)
	reqOK.Header.Set("Authorization", authz)
	recOK := httptest.NewRecorder()
	h.ServeHTTP(recOK, reqOK)
	if recOK.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", recOK.Code, recOK.Body.String())
	}
	var search struct {
		Members []oas.MemberDirectoryEntry `json:"members"`
	}
	if err := json.Unmarshal(recOK.Body.Bytes(), &search); err != nil {
		t.Fatalf("decode: %v", err)
	}
	found := false
	for _, m := range search.Members {
		if m.MemberId == "m3" {
			found = true
		}
	}
	if !found {
		t.Fatalf("expected to find m3 in results, members=%v", search.Members)
	}
}

func TestMembers_VehicleProfile_CreateGetAndPatch(t *testing.T) {
	t.Parallel()

	h, mint := newTestMemberRouter(t)
	authz := "Bearer " + mint(time.Unix(1700000000, 0), "kid-1")

	// Create with vehicleProfile.
	createBody := `{
		"displayName":"Alice",
		"email":"alice@example.com",
		"vehicleProfile":{"make":"Toyota","model":"4Runner","notes":"hello"}
	}`
	req := httptest.NewRequest(http.MethodPost, "/members", bytes.NewBufferString(createBody))
	req.Header.Set("Authorization", authz)
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusCreated {
		t.Fatalf("create status=%d body=%s", rec.Code, rec.Body.String())
	}

	// Get should include the vehicle profile.
	reqGet := httptest.NewRequest(http.MethodGet, "/members/me", nil)
	reqGet.Header.Set("Authorization", authz)
	recGet := httptest.NewRecorder()
	h.ServeHTTP(recGet, reqGet)
	if recGet.Code != http.StatusOK {
		t.Fatalf("get status=%d body=%s", recGet.Code, recGet.Body.String())
	}
	var got struct {
		Member oas.MemberProfile `json:"member"`
	}
	if err := json.Unmarshal(recGet.Body.Bytes(), &got); err != nil {
		t.Fatalf("decode get: %v", err)
	}
	if got.Member.VehicleProfile == nil {
		t.Fatalf("expected vehicleProfile present")
	}

	// Patch vehicle profile fields (null + value).
	patchBody := `{"vehicleProfile":{"make":null,"model":"Tacoma"}}`
	reqPatch := httptest.NewRequest(http.MethodPatch, "/members/me", bytes.NewBufferString(patchBody))
	reqPatch.Header.Set("Authorization", authz)
	reqPatch.Header.Set("Content-Type", "application/json")
	reqPatch.Header.Set("Idempotency-Key", "idem-vp-1")
	recPatch := httptest.NewRecorder()
	h.ServeHTTP(recPatch, reqPatch)
	if recPatch.Code != http.StatusOK {
		t.Fatalf("patch status=%d body=%s", recPatch.Code, recPatch.Body.String())
	}
}
