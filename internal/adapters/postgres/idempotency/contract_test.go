package idempotency

import (
	"testing"

	"eastbay-overland-rally-planner/internal/adapters/contracttest"
	"eastbay-overland-rally-planner/internal/adapters/postgres/testutil"
	idempotencyport "eastbay-overland-rally-planner/internal/ports/out/idempotency"
)

func TestContract_PostgresIdempotencyStore(t *testing.T) {
	pool := testutil.OpenMigratedPool(t)
	issuer := "https://issuer.test"

	contracttest.RunIdempotencyStore(t, func(t *testing.T) (idempotencyport.Store, func()) {
		t.Helper()
		return NewStore(pool, issuer), nil
	})
}
