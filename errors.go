package playwell

import (
	"errors"
	"fmt"
)

var (
	// ErrConnection indicates network failure or timeout.
	ErrConnection = errors.New("playwell: connection failed")

	// ErrVersionMismatch indicates the server does not support client's API version.
	ErrVersionMismatch = errors.New("playwell: unsupported API version")

	// ErrNotFound indicates the requested kit or instruction manual does not exist on the server (HTTP 404).
	ErrNotFound = errors.New("playwell: resource not found")
)

// APIError represents an unexpected 4xx / 5xx response from server
type APIError struct {
	StatusCode int
	Detail     string
}

func (e *APIError) Error() string {
	return fmt.Sprintf("playwell: HTTP %d: %s", e.StatusCode, e.Detail)
}

// Allows error.Is(err, ErrNotFound) to work seemlessly when StatusCode == 404.
func (e *APIError) Unwrap() error {
	if e.StatusCode == 404 {
		return ErrNotFound
	}
	return nil
}
