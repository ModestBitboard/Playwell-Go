package playwell

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"slices"
	"strings"
	"sync"
	"time"
)

const (
	DefaultServerURL = "https://playwell.bitboard.ca"
	APIVersion       = "v1"
	userAgent        = "playwell-go/1.0.0"
)

// Client is a synchronous HTTP client for Playwell REST API.
type Client struct {
	baseURL       string
	apiVersion    string
	httpClient    *http.Client
	verifyVersion bool

	mu           sync.RWMutex
	serverStatus *ServerStatus
}

// Option configures a Client during init.
type Option func(*Client)

// WithBaseURL overrides the default Playwell server URL.
func WithBaseURL(url string) Option {
	return func(c *Client) {
		if trimmed := strings.TrimSpace(url); trimmed != "" {
			c.baseURL = strings.TrimRight(trimmed, "/")
		}
	}
}

// WithTimeout configures the timeout on the underlying HTTP client.
func WithTimeout(timeout time.Duration) Option {
	return func(c *Client) {
		c.httpClient.Timeout = timeout
	}
}

// WithHTTPClient allows providing a custom *http.Client.
func WithHTTPClient(hc *http.Client) Option {
	return func(c *Client) {
		if hc != nil {
			c.httpClient = hc
		}
	}
}

// WithoutVersionCheck skips the startup handshake compatability check.
func WithoutVersionCheck() Option {
	return func(c *Client) {
		c.verifyVersion = false
	}
}

/*
NewClient initalizes a new Playwell API client.
By default, it resolves PLAYWELL_SERVER_URL or falls back to https://playwell.bitboard.ca,
and verifies server compatibility unless WithoutVersionChecK() is passed.
*/
func NewClient(opts ...Option) (*Client, error) {
	resolvedURL := DefaultServerURL
	if envURL := strings.TrimSpace(os.Getenv("PLAYWELL_SERVER_URL")); envURL != "" {
		resolvedURL = strings.TrimRight(envURL, "/")
	}

	c := &Client{
		baseURL:       resolvedURL,
		apiVersion:    APIVersion,
		httpClient:    &http.Client{Timeout: 15 * time.Second},
		verifyVersion: true,
	}

	for _, opt := range opts {
		opt(c)
	}

	if c.verifyVersion {
		if err := c.checkCompatability(context.Background()); err != nil {
			return nil, err
		}
	}

	return c, nil
}

// GetStatus fetches the latest server status handshake from GET / and caches it.
func (c *Client) GetStatus(ctx context.Context) (*ServerStatus, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.baseURL+"/", nil)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrConnection, err)
	}
	req.Header.Set("User-Agent", userAgent)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrConnection, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		body, _ := io.ReadAll(resp.Body)
		return nil, &APIError{
			StatusCode: resp.StatusCode,
			Detail:     strings.TrimSpace(string(body)),
		}
	}

	var status ServerStatus
	if err := json.NewDecoder(resp.Body).Decode(&status); err != nil {
		return nil, fmt.Errorf("playwell: invalid handshake response from %s: %w", c.baseURL, err)
	}

	c.mu.Lock()
	c.serverStatus = &status
	c.mu.Unlock()

	return &status, nil
}

// ServerStatus returns the cached ServerStatus object, fetching it automatically if not yet populated.
func (c *Client) ServerStatus(ctx context.Context) (*ServerStatus, error) {
	c.mu.RLock()
	cached := c.serverStatus
	c.mu.RUnlock()

	if cached != nil {
		return cached, nil
	}

	return c.GetStatus(ctx)
}

func (c *Client) checkCompatability(ctx context.Context) error {
	status, err := c.GetStatus(ctx)
	if err != nil {
		return err
	}

	if len(status.AvailableAPIVersions) == 0 {
		return fmt.Errorf("playwell: invalid handshake response from %s: missing available_api_versions", c.baseURL)
	}

	if !slices.Contains(status.AvailableAPIVersions, c.apiVersion) {
		return fmt.Errorf(
			"%w: server at %s does not support API %q (supported: %s)",
			ErrVersionMismatch,
			c.baseURL,
			c.apiVersion,
			strings.Join(status.AvailableAPIVersions, ", "),
		)
	}

	return nil
}
