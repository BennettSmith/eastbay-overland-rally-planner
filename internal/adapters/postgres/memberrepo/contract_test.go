package memberrepo

import (
	"testing"

	"github.com/BennettSmith/ebo-planner-backend/internal/adapters/contracttest"
	"github.com/BennettSmith/ebo-planner-backend/internal/adapters/postgres/testutil"
	memberrepoport "github.com/BennettSmith/ebo-planner-backend/internal/ports/out/memberrepo"
)

func TestContract_PostgresMemberRepo(t *testing.T) {
	pool := testutil.OpenMigratedPool(t)
	issuer := "https://issuer.test"

	contracttest.RunMemberRepo(t, func(t *testing.T) (memberrepoport.Repository, func()) {
		t.Helper()
		return NewRepo(pool, issuer), nil
	})
}
