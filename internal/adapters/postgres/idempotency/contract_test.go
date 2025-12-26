package idempotency

import (
	"testing"

	"github.com/BennettSmith/ebo-planner-backend/internal/adapters/contracttest"
	"github.com/BennettSmith/ebo-planner-backend/internal/adapters/postgres/testutil"
	idempotencyport "github.com/BennettSmith/ebo-planner-backend/internal/ports/out/idempotency"
)

func TestContract_PostgresIdempotencyStore(t *testing.T) {
	pool := testutil.OpenMigratedPool(t)
	issuer := "https://issuer.test"

	contracttest.RunIdempotencyStore(t, func(t *testing.T) (idempotencyport.Store, func()) {
		t.Helper()
		return NewStore(pool, issuer), nil
	})
}
