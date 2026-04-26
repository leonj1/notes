package sdk

import (
	"fmt"
	"strings"
)

// APIError represents a non-2xx response from the notes service. The server
// returns plain-text error bodies (see USAGE.md), which we surface verbatim
// alongside the HTTP status code.
type APIError struct {
	StatusCode int
	Status     string
	Body       string
}

// Error implements the error interface.
func (e *APIError) Error() string {
	body := strings.TrimSpace(e.Body)
	if body == "" {
		return fmt.Sprintf("notes sdk: unexpected status %s", e.Status)
	}
	return fmt.Sprintf("notes sdk: unexpected status %s: %s", e.Status, body)
}

// IsNotFound reports whether the error is an HTTP 404 from the server.
func (e *APIError) IsNotFound() bool { return e.StatusCode == 404 }

// IsBadRequest reports whether the error is an HTTP 400 from the server.
func (e *APIError) IsBadRequest() bool { return e.StatusCode == 400 }
