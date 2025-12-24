package members

import (
	"fmt"
)

// Error is an application-layer error that can be mapped to an HTTP/OpenAPI error response.
type Error struct {
	Status  int
	Code    string
	Message string
	Details map[string]any
}

func (e *Error) Error() string {
	if e == nil {
		return "<nil>"
	}
	if e.Code == "" {
		return fmt.Sprintf("app error (status=%d): %s", e.Status, e.Message)
	}
	return fmt.Sprintf("%s: %s", e.Code, e.Message)
}

func (e *Error) WithDetails(details map[string]any) *Error {
	if e == nil {
		return nil
	}
	// Copy to avoid accidental shared mutation.
	cp := make(map[string]any, len(details))
	for k, v := range details {
		cp[k] = v
	}
	out := *e
	out.Details = cp
	return &out
}
