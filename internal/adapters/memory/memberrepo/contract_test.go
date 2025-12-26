package memberrepo

import (
	"testing"

	"github.com/BennettSmith/ebo-planner-backend/internal/adapters/contracttest"
	memberrepoport "github.com/BennettSmith/ebo-planner-backend/internal/ports/out/memberrepo"
)

func TestContract_MemberRepo(t *testing.T) {
	contracttest.RunMemberRepo(t, func(t *testing.T) (memberrepoport.Repository, func()) {
		t.Helper()
		return NewRepo(), nil
	})
}
