package retry

import (
	"time"

	"github.com/avast/retry-go/v4"
)

const (
	defaultAttempts = 3
	defaultMaxDelay = 2 * time.Second
	defaultDelay    = 100 * time.Millisecond
)

type RetryConfig struct {
	Attempts uint          `env:"ATTEMPTS,notEmpty"`
	Delay    time.Duration `env:"DELAY,notEmpty"`
	MaxDelay time.Duration `env:"MAX_DELAY,notEmpty"`
	Timeout  time.Duration `env:"TIMEOUT,notEmpty"`
}

func (rc *RetryConfig) ToRetryOptions() []retry.Option {
	return []retry.Option{
		retry.Attempts(rc.Attempts),
		retry.MaxDelay(rc.MaxDelay),
		retry.Delay(rc.Delay),
	}
}

func DefaultRetryConfig() *RetryConfig {
	return &RetryConfig{
		Attempts: defaultAttempts,
		Delay:    defaultMaxDelay,
		MaxDelay: defaultDelay,
	}
}
