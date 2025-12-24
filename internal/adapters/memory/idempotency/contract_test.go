package idempotency

import (
	"testing"

	"eastbay-overland-rally-planner/internal/adapters/contracttest"
	idempotencyport "eastbay-overland-rally-planner/internal/ports/out/idempotency"
)

func TestContract_IdempotencyStore(t *testing.T) {
	contracttest.RunIdempotencyStore(t, func(t *testing.T) (idempotencyport.Store, func()) {
		t.Helper()
		return NewStore(), nil
	})
}
