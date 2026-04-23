package api

import "fmt"

// APIError is returned by the API client for non-2xx HTTP responses.
// Callers can use errors.As to inspect StatusCode for tip lookup or JSON error codes.
type APIError struct {
	StatusCode int
	Message    string
}

func (e *APIError) Error() string {
	return fmt.Sprintf("HTTP %d: %s", e.StatusCode, e.Message)
}
