package sdk

import (
	"context"
	"math"
	"math/rand"
	"time"
)

// ReconnectConfig controls automatic reconnection behavior.
type ReconnectConfig struct {
	InitialDelay time.Duration // Default: 1s
	MaxDelay     time.Duration // Default: 30s
	Multiplier   float64       // Default: 2.0
	Jitter       float64       // Default: 0.2 (±20%)
	MaxAttempts  int           // 0 = unlimited
}

func DefaultReconnectConfig() ReconnectConfig {
	return ReconnectConfig{
		InitialDelay: 1 * time.Second,
		MaxDelay:     30 * time.Second,
		Multiplier:   2.0,
		Jitter:       0.2,
		MaxAttempts:  0,
	}
}

// ConnectWithReconnect tries to connect with exponential backoff.
func (c *Client) ConnectWithReconnect(ctx context.Context) error {
	cfg := c.config.Reconnect
	if cfg.InitialDelay == 0 {
		cfg = DefaultReconnectConfig()
	}

	delay := cfg.InitialDelay
	attempt := 0

	for {
		err := c.Connect(ctx)
		if err == nil {
			return nil
		}

		attempt++
		if cfg.MaxAttempts > 0 && attempt >= cfg.MaxAttempts {
			return err
		}

		jittered := applyJitter(delay, cfg.Jitter)

		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(jittered):
		}

		delay = time.Duration(float64(delay) * cfg.Multiplier)
		if delay > cfg.MaxDelay {
			delay = cfg.MaxDelay
		}
	}
}

func applyJitter(d time.Duration, jitter float64) time.Duration {
	if jitter <= 0 {
		return d
	}
	delta := float64(d) * jitter
	min := float64(d) - delta
	max := float64(d) + delta
	return time.Duration(min + rand.Float64()*(max-min))
}

// NextDelay computes the delay for a given attempt (exported for testing).
func NextDelay(attempt int, cfg ReconnectConfig) time.Duration {
	delay := float64(cfg.InitialDelay) * math.Pow(cfg.Multiplier, float64(attempt))
	if delay > float64(cfg.MaxDelay) {
		delay = float64(cfg.MaxDelay)
	}
	return applyJitter(time.Duration(delay), cfg.Jitter)
}
