// Package httpapi is the HTTP transport adapter (ports/adapters "delivery" layer).
//
// It should depend on:
// - the generated OpenAPI transport package: internal/adapters/httpapi/oas
// - your application layer ports: internal/ports
//
// It should NOT be imported by internal/app or internal/domain.
package httpapi
