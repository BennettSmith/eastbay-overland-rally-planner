package httpapi

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Overland-East-Bay/trip-planner-api/internal/adapters/httpapi/oas"
)

func TestStrictUnimplemented_ReturnsOpenAPI500WithNotImplementedError(t *testing.T) {
	t.Parallel()

	s := StrictUnimplemented{}
	ctx := context.Background()

	resp, err := s.ListMembers(ctx, oas.ListMembersRequestObject{})
	if err != nil {
		t.Fatalf("err=%v", err)
	}
	w := httptest.NewRecorder()
	if err := resp.VisitListMembersResponse(w); err != nil {
		t.Fatalf("visit: %v", err)
	}
	if w.Code != http.StatusInternalServerError {
		t.Fatalf("status=%d", w.Code)
	}

	// Spot-check a few more endpoints to ensure wiring is consistent.
	resp2, err := s.CreateMyMember(ctx, oas.CreateMyMemberRequestObject{})
	if err != nil {
		t.Fatalf("err2=%v", err)
	}
	w2 := httptest.NewRecorder()
	if err := resp2.VisitCreateMyMemberResponse(w2); err != nil {
		t.Fatalf("visit2: %v", err)
	}
	if w2.Code != http.StatusInternalServerError {
		t.Fatalf("status2=%d", w2.Code)
	}

	resp3, err := s.PublishTrip(ctx, oas.PublishTripRequestObject{TripId: "t1"})
	if err != nil {
		t.Fatalf("err3=%v", err)
	}
	w3 := httptest.NewRecorder()
	if err := resp3.VisitPublishTripResponse(w3); err != nil {
		t.Fatalf("visit3: %v", err)
	}
	if w3.Code != http.StatusInternalServerError {
		t.Fatalf("status3=%d", w3.Code)
	}
}
