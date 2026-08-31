package usecase

import (
	"log/slog"
	"time"
)

// RetryConfig controls how transient failures in external calls (platform
// API, AI reviewer) are retried before giving up on a pipeline step.
type RetryConfig struct {
	// MaxAttempts is the total number of tries (including the first one).
	// Values below 1 are treated as 1 (no retry).
	MaxAttempts int
	// BaseDelay is the wait before the second attempt; each subsequent
	// attempt doubles it (exponential backoff).
	BaseDelay time.Duration
}

// DefaultRetryConfig matches the documented default: 3 attempts, 2s base delay.
func DefaultRetryConfig() RetryConfig {
	return RetryConfig{MaxAttempts: 3, BaseDelay: 2 * time.Second}
}

// run executes fn, retrying on error up to MaxAttempts times with exponential
// backoff between tries. Returns the last error when every attempt fails.
func (c RetryConfig) run(log *slog.Logger, op string, fn func() error) error {
	attempts := c.MaxAttempts
	if attempts < 1 {
		attempts = 1
	}

	var err error
	for attempt := 1; attempt <= attempts; attempt++ {
		if err = fn(); err == nil {
			return nil
		}
		if attempt == attempts {
			break
		}
		delay := c.BaseDelay * time.Duration(1<<(attempt-1))
		log.Warn("retrying after transient error",
			slog.String("op", op),
			slog.Int("attempt", attempt),
			slog.Int("max_attempts", attempts),
			slog.Duration("delay", delay),
			slog.String("error", err.Error()),
		)
		time.Sleep(delay)
	}
	return err
}
