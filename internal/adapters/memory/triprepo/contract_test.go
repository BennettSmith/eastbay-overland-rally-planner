package triprepo

import (
	"testing"

	"eastbay-overland-rally-planner/internal/adapters/contracttest"
	memmemberrepo "eastbay-overland-rally-planner/internal/adapters/memory/memberrepo"
	memrsvprepo "eastbay-overland-rally-planner/internal/adapters/memory/rsvprepo"
	memberrepoport "eastbay-overland-rally-planner/internal/ports/out/memberrepo"
	rsvprepoport "eastbay-overland-rally-planner/internal/ports/out/rsvprepo"
	triprepoport "eastbay-overland-rally-planner/internal/ports/out/triprepo"
)

func TestContract_TripAndRSVPRepos(t *testing.T) {
	contracttest.RunTripAndRSVPRepos(
		t,
		func(t *testing.T) (memberrepoport.Repository, func()) {
			t.Helper()
			return memmemberrepo.NewRepo(), nil
		},
		func(t *testing.T) (triprepoport.Repository, func()) {
			t.Helper()
			return NewRepo(), nil
		},
		func(t *testing.T) (rsvprepoport.Repository, func()) {
			t.Helper()
			return memrsvprepo.NewRepo(), nil
		},
	)
}
