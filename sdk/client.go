package sdk

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// userAgent is sent on every request so the server can identify SDK callers.
var userAgent = "notes-go-sdk/" + Version

// Client is the entry point for interacting with the notes service.
//
// Construct one with NewClient and reuse it for the lifetime of your
// application; *Client is safe for concurrent use by multiple goroutines as
// long as the underlying *http.Client is.
type Client struct {
	baseURL    string
	httpClient *http.Client
}

// Option configures a Client at construction time.
type Option func(*Client)

// WithHTTPClient overrides the *http.Client used for all requests. Callers
// can use this to supply custom transports, TLS configuration, or middleware
// (e.g. authentication round-trippers).
func WithHTTPClient(h *http.Client) Option {
	return func(c *Client) {
		if h != nil {
			c.httpClient = h
		}
	}
}

// WithTimeout sets a request timeout on the default *http.Client. It is a
// no-op when WithHTTPClient was used to install a caller-supplied client.
func WithTimeout(d time.Duration) Option {
	return func(c *Client) {
		c.httpClient.Timeout = d
	}
}

// NewClient returns a Client that talks to the notes service rooted at
// baseURL. baseURL may be specified with or without a trailing slash and may
// include a path prefix (e.g. "https://api.example.com/notes").
func NewClient(baseURL string, opts ...Option) *Client {
	c := &Client{
		baseURL:    strings.TrimRight(baseURL, "/"),
		httpClient: &http.Client{Timeout: 30 * time.Second},
	}
	for _, opt := range opts {
		opt(c)
	}
	return c
}

// do performs an HTTP request against the configured base URL and decodes
// the JSON response body into out (when non-nil). Non-2xx responses produce
// an *APIError carrying the server-supplied status and body.
func (c *Client) do(ctx context.Context, method, path string, query url.Values, body, out any) error {
	u := c.baseURL + path
	if len(query) > 0 {
		u += "?" + query.Encode()
	}

	var reqBody io.Reader
	if body != nil {
		buf, err := json.Marshal(body)
		if err != nil {
			return fmt.Errorf("notes sdk: encode request body: %w", err)
		}
		reqBody = bytes.NewReader(buf)
	}

	req, err := http.NewRequestWithContext(ctx, method, u, reqBody)
	if err != nil {
		return fmt.Errorf("notes sdk: build request: %w", err)
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", userAgent)
	if reqBody != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("notes sdk: %s %s: %w", method, u, err)
	}
	defer resp.Body.Close()

	respBody, readErr := io.ReadAll(resp.Body)
	if readErr != nil {
		return fmt.Errorf("notes sdk: read response: %w", readErr)
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return &APIError{
			StatusCode: resp.StatusCode,
			Status:     resp.Status,
			Body:       string(respBody),
		}
	}

	if out == nil || len(respBody) == 0 {
		return nil
	}
	if err := json.Unmarshal(respBody, out); err != nil {
		return fmt.Errorf("notes sdk: decode response: %w", err)
	}
	return nil
}
