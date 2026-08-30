package middleware

import (
	"net/http"
	"time"

	"github.com/juju/ratelimit"

	"github.com/gohade/hade/framework/gin"
)

var defaultRateLimit = RateLimit{
	cap:             1000,
	rate:            500,
	waitMaxDuration: 0,
}

// RateLimit ...
type RateLimit struct {
	cap             int64
	rate            float64
	waitMaxDuration time.Duration
	bucket          *ratelimit.Bucket
}

// RateLimitOption ...
type RateLimitOption func(r *RateLimit)

// WithRate set rate
func WithRate(rate float64) RateLimitOption {
	return func(r *RateLimit) {
		r.rate = rate
	}
}

// WithCap set cap
func WithCap(cap int64) RateLimitOption {
	return func(r *RateLimit) {
		r.cap = cap
	}
}

// WithWaitMaxDuration set waitMaxDuration
func WithWaitMaxDuration(max time.Duration) RateLimitOption {
	return func(r *RateLimit) {
		r.waitMaxDuration = max
	}
}

// NewRateLimit ...
func NewRateLimit(opts ...RateLimitOption) *RateLimit {
	r := defaultRateLimit
	for _, opt := range opts {
		opt(&r)
	}

	r.bucket = ratelimit.NewBucketWithRate(r.rate, r.cap)

	return &r
}

// Func ...
func (r *RateLimit) Func() gin.HandlerFunc {
	return func(c *gin.Context) {
		if _, ok := r.bucket.TakeMaxDuration(1, r.waitMaxDuration); !ok {
			c.JSON(http.StatusTooManyRequests, nil)
			return
		}

		c.Next()
	}
}
