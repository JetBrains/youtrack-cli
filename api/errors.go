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

// UnexpectedResponseError is returned when an API endpoint responds with a
// payload that cannot be decoded as the expected YouTrack API JSON.
type UnexpectedResponseError struct {
	StatusCode  int
	ContentType string
	Err         error
}

func (e *UnexpectedResponseError) Error() string {
	contentType := e.ContentType
	if contentType == "" {
		contentType = "unknown content type"
	}
	return fmt.Sprintf("unexpected YouTrack API response (HTTP %d, %s): %v", e.StatusCode, contentType, e.Err)
}
