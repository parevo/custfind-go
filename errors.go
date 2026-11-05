package custfind

import (
	"errors"
	"fmt"
	"net/http"
	"time"
)

// Error codes the API returns in {"error": "<code>"}.
const (
	CodeMissingAPIKey          = "missing_api_key"
	CodeInvalidAPIKey          = "invalid_api_key"
	CodeAPIRequiresPaidPlan    = "api_requires_paid_plan"
	CodeInsufficientScope      = "insufficient_scope"
	CodeCustomerLimitReached   = "customer_limit_reached"
	CodeRateLimited            = "rate_limited"
	CodeIdempotencyKeyReused   = "idempotency_key_reused"
	CodeIdempotencyInProgress  = "idempotency_in_progress"
	CodeInvalidIdempotencyKey  = "invalid_idempotency_key"
	CodeAIUnavailable          = "ai_unavailable"
	CodeResponseTooLargeReplay = "response_too_large_to_replay"
)

// Error is an error response from the API, or a request that never got one
// (StatusCode 0, Code "network_error").
type Error struct {
	// StatusCode is the HTTP status, or 0 when there was no response.
	StatusCode int
	// Code is the API's error value, e.g. "customer_limit_reached", or
	// "http_<status>" when the body had none.
	Code string
	// Message holds extra detail when the API gave some.
	Message string
	// RetryAfter is the server's Retry-After, if it sent one.
	RetryAfter time.Duration

	err error
}

func (e *Error) Error() string {
	if e.StatusCode == 0 {
		return fmt.Sprintf("custfind: request failed: %s", e.Message)
	}
	if e.Message != "" {
		return fmt.Sprintf("custfind: API error %d: %s (%s)", e.StatusCode, e.Code, e.Message)
	}
	return fmt.Sprintf("custfind: API error %d: %s", e.StatusCode, e.Code)
}

func (e *Error) Unwrap() error { return e.err }

// retryable reports whether the same request may succeed when sent again.
// A 409 is retried only while an earlier attempt of the same idempotent write
// is still running; any other 409 ("invalid job state") is final and would
// only be replayed.
func (e *Error) retryable() bool {
	switch {
	case e.StatusCode == http.StatusTooManyRequests, e.StatusCode >= 500:
		return true
	case e.StatusCode == http.StatusConflict:
		return e.Code == CodeIdempotencyInProgress
	}
	return false
}

// IsNotFound reports whether err is a 404 from the API.
func IsNotFound(err error) bool { return hasStatus(err, http.StatusNotFound) }

// IsPaymentRequired reports whether err is a 402: no credits left, a
// customer's monthly cap reached, or a plan without API access.
func IsPaymentRequired(err error) bool { return hasStatus(err, http.StatusPaymentRequired) }

// HasCode reports whether err is an API error with the given code.
func HasCode(err error, code string) bool {
	var e *Error
	return errors.As(err, &e) && e.Code == code
}

func hasStatus(err error, status int) bool {
	var e *Error
	return errors.As(err, &e) && e.StatusCode == status
}
