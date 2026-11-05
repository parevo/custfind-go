// Package custfind is the official Go client for the Custfind API: start
// lead searches, read scored leads, write and send emails, move deals and
// receive webhooks.
//
//	cf, err := custfind.NewClient() // reads CUSTFIND_API_KEY
//	created, err := cf.Searches.Create(ctx, custfind.CreateSearchParams{
//		Sector: "Dental clinic", Location: "Kadıköy, Istanbul", Limit: 25,
//	})
//
// Every write is sent with an Idempotency-Key that stays the same across its
// retries, so a retried search never starts twice or reserves credits twice.
package custfind

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	mrand "math/rand/v2"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"
)

// Version is the SDK version, sent in the User-Agent.
const Version = "1.0.0"

const (
	DefaultBaseURL    = "https://api.custfind.com"
	defaultTimeout    = 30 * time.Second
	defaultMaxRetries = 3
)

// Client talks to the Custfind /v1 API. It is safe for concurrent use.
type Client struct {
	apiKey     string
	baseURL    string
	httpClient *http.Client
	maxRetries int
	userAgent  string

	Searches       *SearchesService
	Leads          *LeadsService
	Outreach       *OutreachService
	Pipelines      *PipelinesService
	Deals          *DealsService
	Webhooks       *WebhooksService
	Customers      *CustomersService
	SenderAccounts *SenderAccountsService
}

// Option configures a Client.
type Option func(*Client)

// WithAPIKey sets the workspace key (cf_live_…). Defaults to CUSTFIND_API_KEY.
func WithAPIKey(key string) Option { return func(c *Client) { c.apiKey = key } }

// WithBaseURL points the client at another API host.
func WithBaseURL(u string) Option { return func(c *Client) { c.baseURL = strings.TrimRight(u, "/") } }

// WithHTTPClient sets the http.Client used for requests. Its Timeout is the
// per-attempt timeout.
func WithHTTPClient(hc *http.Client) Option { return func(c *Client) { c.httpClient = hc } }

// WithTimeout sets the per-attempt timeout. Default 30 s.
func WithTimeout(d time.Duration) Option {
	return func(c *Client) {
		hc := *c.httpClient
		hc.Timeout = d
		c.httpClient = &hc
	}
}

// WithMaxRetries sets how often a request is retried after a network error,
// 429, 5xx or an idempotent write still in progress. Default 3.
func WithMaxRetries(n int) Option { return func(c *Client) { c.maxRetries = n } }

// WithUserAgent prepends your own product to the User-Agent.
func WithUserAgent(ua string) Option {
	return func(c *Client) { c.userAgent = ua + " " + c.userAgent }
}

// NewClient returns a client. Without WithAPIKey it reads CUSTFIND_API_KEY.
func NewClient(opts ...Option) (*Client, error) {
	c := &Client{
		apiKey:     os.Getenv("CUSTFIND_API_KEY"),
		baseURL:    DefaultBaseURL,
		httpClient: &http.Client{Timeout: defaultTimeout},
		maxRetries: defaultMaxRetries,
		userAgent:  "custfind-go/" + Version,
	}
	for _, opt := range opts {
		opt(c)
	}
	if c.apiKey == "" {
		return nil, errors.New("custfind: pass WithAPIKey or set CUSTFIND_API_KEY")
	}
	c.Searches = &SearchesService{c}
	c.Leads = &LeadsService{c}
	c.Outreach = &OutreachService{c}
	c.Pipelines = &PipelinesService{c}
	c.Deals = &DealsService{c}
	c.Webhooks = &WebhooksService{c}
	c.Customers = &CustomersService{c}
	c.SenderAccounts = &SenderAccountsService{c}
	return c, nil
}

// RequestOption changes a single call.
type RequestOption func(*requestConfig)

type requestConfig struct {
	idempotencyKey string
}

// WithIdempotencyKey sends your own Idempotency-Key with a write instead of a
// generated one, which makes a retry across process restarts safe too. The
// API remembers a key for 24 hours.
func WithIdempotencyKey(key string) RequestOption {
	return func(rc *requestConfig) { rc.idempotencyKey = key }
}

// Me is the workspace and key behind the current API key.
type Me struct {
	OrganizationID string `json:"organization_id"`
	APIKeyID       string `json:"api_key_id"`
}

// Me returns the workspace and key behind the current key.
func (c *Client) Me(ctx context.Context) (*Me, error) {
	var out Me
	return &out, c.do(ctx, http.MethodGet, "/v1/me", nil, nil, &out)
}

// Balance returns the plan and remaining credits.
func (c *Client) Balance(ctx context.Context) (*Balance, error) {
	var out Balance
	return &out, c.do(ctx, http.MethodGet, "/v1/balance", nil, nil, &out)
}

// Usage returns what API searches spent, optionally grouped by customer or key.
func (c *Client) Usage(ctx context.Context, params UsageParams) (*Usage, error) {
	var out Usage
	return &out, c.do(ctx, http.MethodGet, "/v1/usage", encodeQuery(params), nil, &out)
}

// do sends one API call with retries and decodes a JSON response into out
// (which may be nil).
func (c *Client) do(ctx context.Context, method, path string, query url.Values, body any, out any, opts ...RequestOption) error {
	var rc requestConfig
	for _, opt := range opts {
		opt(&rc)
	}
	u := c.baseURL + path
	if len(query) > 0 {
		u += "?" + query.Encode()
	}
	var payload []byte
	if body != nil {
		var err error
		if payload, err = json.Marshal(body); err != nil {
			return fmt.Errorf("custfind: encode request: %w", err)
		}
	}
	idemKey := ""
	if method != http.MethodGet {
		idemKey = rc.idempotencyKey
		if idemKey == "" {
			idemKey = newIdempotencyKey()
		}
	}

	for attempt := 0; ; attempt++ {
		req, err := http.NewRequestWithContext(ctx, method, u, bytes.NewReader(payload))
		if err != nil {
			return fmt.Errorf("custfind: %w", err)
		}
		if payload == nil {
			req.Body, req.ContentLength = http.NoBody, 0
		} else {
			req.Header.Set("Content-Type", "application/json")
		}
		req.Header.Set("Authorization", "Bearer "+c.apiKey)
		req.Header.Set("Accept", "application/json")
		req.Header.Set("User-Agent", c.userAgent)
		if idemKey != "" {
			req.Header.Set("Idempotency-Key", idemKey)
		}

		resp, err := c.httpClient.Do(req)
		if err != nil {
			if ctx.Err() != nil {
				return ctx.Err()
			}
			if attempt < c.maxRetries {
				if err := sleep(ctx, backoff(attempt)); err != nil {
					return err
				}
				continue
			}
			return &Error{Code: "network_error", Message: err.Error(), err: err}
		}

		if resp.StatusCode >= 200 && resp.StatusCode < 300 {
			defer resp.Body.Close()
			if out == nil || resp.StatusCode == http.StatusNoContent {
				_, _ = io.Copy(io.Discard, resp.Body)
				return nil
			}
			data, err := io.ReadAll(resp.Body)
			if err != nil {
				return fmt.Errorf("custfind: read response: %w", err)
			}
			if len(bytes.TrimSpace(data)) == 0 {
				return nil
			}
			if err := json.Unmarshal(data, out); err != nil {
				return fmt.Errorf("custfind: decode response: %w", err)
			}
			return nil
		}

		apiErr := readError(resp)
		if apiErr.retryable() && attempt < c.maxRetries {
			wait := apiErr.RetryAfter
			if wait <= 0 {
				wait = backoff(attempt)
			}
			if err := sleep(ctx, wait); err != nil {
				return err
			}
			continue
		}
		return apiErr
	}
}

func readError(resp *http.Response) *Error {
	defer resp.Body.Close()
	e := &Error{StatusCode: resp.StatusCode, Code: fmt.Sprintf("http_%d", resp.StatusCode)}
	if s, err := strconv.Atoi(resp.Header.Get("Retry-After")); err == nil && s >= 0 {
		e.RetryAfter = time.Duration(s) * time.Second
	}
	data, _ := io.ReadAll(io.LimitReader(resp.Body, 64<<10))
	var body struct {
		Error  string `json:"error"`
		Detail string `json:"detail"`
	}
	if json.Unmarshal(data, &body) == nil && body.Error != "" {
		e.Code = body.Error
		e.Message = body.Detail
	} else {
		// A few routes answer in plain text ("invalid json body").
		e.Message = strings.TrimSpace(string(data))
	}
	return e
}

func newIdempotencyKey() string {
	var b [16]byte
	_, _ = rand.Read(b[:])
	return hex.EncodeToString(b[:])
}

func backoff(attempt int) time.Duration {
	d := 500 * time.Millisecond << attempt
	if d > 8*time.Second || d <= 0 {
		d = 8 * time.Second
	}
	return d + time.Duration(mrand.IntN(250))*time.Millisecond
}

func sleep(ctx context.Context, d time.Duration) error {
	t := time.NewTimer(d)
	defer t.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-t.C:
		return nil
	}
}

func pathEscape(s string) string { return url.PathEscape(s) }

// Ptr returns a pointer to v, for optional fields: custfind.Ptr(5000).
func Ptr[T any](v T) *T { return &v }
