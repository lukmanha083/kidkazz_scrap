package stealth

import (
	"context"
	"math/rand/v2"
	"time"
)

// DelayProfile defines a named delay configuration.
type DelayProfile string

const (
	ProfileCautious   DelayProfile = "cautious"
	ProfileNormal     DelayProfile = "normal"
	ProfileAggressive DelayProfile = "aggressive"
)

// HumanDelay adds randomized jitter to mimic human browsing patterns.
type HumanDelay struct {
	MinDelay time.Duration
	MaxDelay time.Duration
}

// NewHumanDelay creates a delay generator for the given profile.
func NewHumanDelay(profile DelayProfile) *HumanDelay {
	switch profile {
	case ProfileCautious:
		return &HumanDelay{MinDelay: 2 * time.Second, MaxDelay: 5 * time.Second}
	case ProfileAggressive:
		return &HumanDelay{MinDelay: 200 * time.Millisecond, MaxDelay: 800 * time.Millisecond}
	default: // normal
		return &HumanDelay{MinDelay: 500 * time.Millisecond, MaxDelay: 2 * time.Second}
	}
}

// Wait sleeps for a random duration within the configured range.
func (h *HumanDelay) Wait(ctx context.Context) error {
	d := h.RequestDelay()
	select {
	case <-time.After(d):
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

// RequestDelay returns a random delay for API/page requests.
func (h *HumanDelay) RequestDelay() time.Duration {
	return h.randomBetween(h.MinDelay, h.MaxDelay)
}

// PageBrowseDelay returns a longer delay for between-page navigation.
func (h *HumanDelay) PageBrowseDelay() time.Duration {
	return h.randomBetween(h.MaxDelay, h.MaxDelay*2)
}

func (h *HumanDelay) randomBetween(min, max time.Duration) time.Duration {
	if min >= max {
		return min
	}
	return min + time.Duration(rand.Int64N(int64(max-min)))
}
