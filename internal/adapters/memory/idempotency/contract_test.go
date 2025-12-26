package idempotency

import (
	"testing"

	"github.com/BennettSmith/ebo-planner-backend/internal/adapters/contracttest"
	idempotencyport "github.com/BennettSmith/ebo-planner-backend/internal/ports/out/idempotency"
)

func TestContract_IdempotencyStore(t *testing.T) {
	contracttest.RunIdempotencyStore(t, func(t *testing.T) (idempotencyport.Store, func()) {
		t.Helper()
		return NewStore(), nil
	})
}
