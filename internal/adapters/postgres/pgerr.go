package postgres

import (
	"errors"

	"github.com/jackc/pgx/v5/pgconn"
)

const (
	// UniqueViolationCode indicates a unique constraint violation.
	UniqueViolationCode = "23505"
	// ForeignKeyViolationCode indicates a foreign key violation.
	ForeignKeyViolationCode = "23503"
	// CheckViolationCode indicates a check constraint violation.
	CheckViolationCode = "23514"
)

func AsPgError(err error) (*pgconn.PgError, bool) {
	var pe *pgconn.PgError
	if errors.As(err, &pe) {
		return pe, true
	}
	return nil, false
}
