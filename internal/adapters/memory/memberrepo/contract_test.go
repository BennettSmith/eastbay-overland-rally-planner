package memberrepo

import (
	"testing"

	"eastbay-overland-rally-planner/internal/adapters/contracttest"
	memberrepoport "eastbay-overland-rally-planner/internal/ports/out/memberrepo"
)

func TestContract_MemberRepo(t *testing.T) {
	contracttest.RunMemberRepo(t, func(t *testing.T) (memberrepoport.Repository, func()) {
		t.Helper()
		return NewRepo(), nil
	})
}
