package triprepo

import (
	"testing"

	"eastbay-overland-rally-planner/internal/adapters/contracttest"
	"eastbay-overland-rally-planner/internal/adapters/postgres/memberrepo"
	"eastbay-overland-rally-planner/internal/adapters/postgres/rsvprepo"
	"eastbay-overland-rally-planner/internal/adapters/postgres/testutil"
	memberrepoport "eastbay-overland-rally-planner/internal/ports/out/memberrepo"
	rsvprepoport "eastbay-overland-rally-planner/internal/ports/out/rsvprepo"
	triprepoport "eastbay-overland-rally-planner/internal/ports/out/triprepo"
)

func TestContract_PostgresTripAndRSVPRepos(t *testing.T) {
	pool := testutil.OpenMigratedPool(t)
	issuer := "https://issuer.test"

	contracttest.RunTripAndRSVPRepos(
		t,
		func(t *testing.T) (memberrepoport.Repository, func()) {
			t.Helper()
			return memberrepo.NewRepo(pool, issuer), nil
		},
		func(t *testing.T) (triprepoport.Repository, func()) {
			t.Helper()
			return NewRepo(pool), nil
		},
		func(t *testing.T) (rsvprepoport.Repository, func()) {
			t.Helper()
			return rsvprepo.NewRepo(pool), nil
		},
	)
}
