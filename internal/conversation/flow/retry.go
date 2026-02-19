package flow

import (
	"context"
	"errors"
	"net"
	"strings"
	"time"
)

// gatewayHTTPError is a typed error returned by postChat and postTriggerSchedule
// when the agent gateway responds with a non-2xx status. Carrying the status code
// lets retry and overflow-recovery logic classify the error without string parsing.
type gatewayHTTPError struct {
	StatusCode int
	Message    string
}

func (e *gatewayHTTPError) Error() string {
	return e.Message
}

// withGatewayRetry retries fn up to 3 total attempts (2 retries) with exponential
// backoff (300 ms, 600 ms). Only transient errors are retried; context cancellation
// and non-retryable HTTP errors are returned immediately.
func withGatewayRetry(ctx context.Context, fn func() error) error {
	const maxAttempts = 3
	baseDelay := 300 * time.Millisecond

	var lastErr error
	for attempt := 0; attempt < maxAttempts; attempt++ {
		if attempt > 0 {
			delay := baseDelay * time.Duration(1<<(attempt-1)) // 300 ms, 600 ms
			select {
			case <-time.After(delay):
			case <-ctx.Done():
				return ctx.Err()
			}
		}

		err := fn()
		if err == nil {
			return nil
		}

		// Context cancelled/timed out — don't retry.
		if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
			return err
		}

		if !isRetryableGatewayError(err) {
			return err
		}
		lastErr = err
	}
	return lastErr
}

// isRetryableGatewayError returns true for transient failures: network errors,
// HTTP 429 (rate limit), and HTTP 5xx server errors.
func isRetryableGatewayError(err error) bool {
	if err == nil {
		return false
	}
	// Network / transport errors (DNS, TCP, TLS, timeout).
	var netErr net.Error
	if errors.As(err, &netErr) {
		return true
	}
	// Typed gateway errors.
	var gErr *gatewayHTTPError
	if errors.As(err, &gErr) {
		return gErr.StatusCode == 429 || gErr.StatusCode >= 500
	}
	return false
}

// isContextOverflowError returns true when the gateway (or LLM provider) rejected
// the request because the prompt exceeded the model's context window.
func isContextOverflowError(err error) bool {
	if err == nil {
		return false
	}
	// Typed gateway error with 422 Unprocessable Entity.
	var gErr *gatewayHTTPError
	if errors.As(err, &gErr) {
		if gErr.StatusCode == 422 {
			return containsOverflowKeyword(gErr.Message)
		}
		// Some providers return 400 for context overflow.
		if gErr.StatusCode == 400 {
			return containsOverflowKeyword(gErr.Message)
		}
	}
	// Fallback: plain error message check (e.g. wrapped errors from the TS gateway).
	return containsOverflowKeyword(err.Error())
}

func containsOverflowKeyword(s string) bool {
	lower := strings.ToLower(s)
	keywords := []string{
		"context_length_exceeded",
		"context window",
		"context length",
		"too many tokens",
		"prompt is too long",
		"prompt too long",
		"maximum context",
		"exceeds the model",
		"input too long",
		"token limit",
		"tokens exceed",
	}
	for _, kw := range keywords {
		if strings.Contains(lower, kw) {
			return true
		}
	}
	return false
}
