package memberrepo

import (
	"testing"

	"eastbay-overland-rally-planner/internal/adapters/contracttest"
	"eastbay-overland-rally-planner/internal/adapters/postgres/testutil"
	memberrepoport "eastbay-overland-rally-planner/internal/ports/out/memberrepo"
)

func TestContract_PostgresMemberRepo(t *testing.T) {
	pool := testutil.OpenMigratedPool(t)
	issuer := "https://issuer.test"

	contracttest.RunMemberRepo(t, func(t *testing.T) (memberrepoport.Repository, func()) {
		t.Helper()
		return NewRepo(pool, issuer), nil
	})
}
