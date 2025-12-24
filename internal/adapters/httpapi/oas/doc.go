// Package oas contains OpenAPI-generated transport types and server glue.
//
// Generated code is written to api.gen.go by:
//
//	make gen-openapi
//
// Keep this package isolated from the rest of the app:
// - Your domain + app layer should not import this package.
// - The HTTP adapter should translate between these generated DTOs and your own app/domain types.
package oas
