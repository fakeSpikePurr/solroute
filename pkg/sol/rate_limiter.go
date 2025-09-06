package sol

import (
	"context"
	"time"

	"golang.org/x/time/rate"
)

// RateLimiter provides rate limiting functionality for RPC calls
type RateLimiter struct {
	limiter *rate.Limiter
}

// NewRateLimiter creates a new rate limiter with the specified requests per second
func NewRateLimiter(requestsPerSecond int) *RateLimiter {
	return &RateLimiter{
		limiter: rate.NewLimiter(rate.Limit(requestsPerSecond), requestsPerSecond),
	}
}

// Wait blocks until the rate limiter allows the request
func (rl *RateLimiter) Wait(ctx context.Context) error {
	return rl.limiter.Wait(ctx)
}

// Allow returns true if the request is allowed without waiting
func (rl *RateLimiter) Allow() bool {
	return rl.limiter.Allow()
}

// Reserve reserves a token and returns a reservation
func (rl *RateLimiter) Reserve() *rate.Reservation {
	return rl.limiter.Reserve()
}

// SetRate updates the rate limiter's rate
func (rl *RateLimiter) SetRate(requestsPerSecond int) {
	rl.limiter.SetLimit(rate.Limit(requestsPerSecond))
	rl.limiter.SetBurst(requestsPerSecond)
}

// GetRate returns the current rate limit
func (rl *RateLimiter) GetRate() int {
	return int(rl.limiter.Limit())
}

// GetBurst returns the current burst size
func (rl *RateLimiter) GetBurst() int {
	return rl.limiter.Burst()
}

// WaitWithTimeout waits for a token with a timeout
func (rl *RateLimiter) WaitWithTimeout(ctx context.Context, timeout time.Duration) error {
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	return rl.Wait(ctx)
}
