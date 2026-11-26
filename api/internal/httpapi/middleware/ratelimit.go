package middleware

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"golang.org/x/time/rate"
)

// Defaults: a sustained 300 requests per minute per IP with bursts of 300.
const (
	defaultRate    = rate.Limit(5)
	defaultBurst   = 300
	defaultIdleTTL = 3 * time.Minute
)

// RateLimitedError reports a request refused by the rate limiter.
type RateLimitedError struct {
	RetryAfter time.Duration
}

func (e *RateLimitedError) Error() string {
	return fmt.Sprintf("rate limited: retry after %s", e.RetryAfter)
}

// RateLimitOptions tunes the limiter. The zero value uses the defaults.
type RateLimitOptions struct {
	Rate    rate.Limit
	Burst   int
	IdleTTL time.Duration
	// Now is the clock; tests inject a manual one.
	Now func() time.Time
}

type visitor struct {
	limiter  *rate.Limiter
	lastSeen time.Time
}

// RateLimiter keeps one token bucket per client IP.
type RateLimiter struct {
	opts     RateLimitOptions
	mu       sync.Mutex
	visitors map[string]*visitor
}

// NewRateLimiter returns a limiter with opts, defaults filled in.
func NewRateLimiter(opts RateLimitOptions) *RateLimiter {
	if opts.Rate == 0 {
		opts.Rate = defaultRate
	}
	if opts.Burst == 0 {
		opts.Burst = defaultBurst
	}
	if opts.IdleTTL == 0 {
		opts.IdleTTL = defaultIdleTTL
	}
	if opts.Now == nil {
		opts.Now = time.Now
	}
	return &RateLimiter{opts: opts, visitors: map[string]*visitor{}}
}

// Middleware refuses a request over the client IP's budget with
// *RateLimitedError (reported with c.Error) and aborts the chain.
func (l *RateLimiter) Middleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		if wait := l.reserve(c.ClientIP()); wait > 0 {
			_ = c.Error(&RateLimitedError{RetryAfter: wait})
			c.Abort()
			return
		}
		c.Next()
	}
}

// reserve takes a token for ip and returns 0, or how long until one is free.
func (l *RateLimiter) reserve(ip string) time.Duration {
	now := l.opts.Now()
	l.mu.Lock()
	v, ok := l.visitors[ip]
	if !ok {
		v = &visitor{limiter: rate.NewLimiter(l.opts.Rate, l.opts.Burst)}
		l.visitors[ip] = v
	}
	v.lastSeen = now
	l.mu.Unlock()

	r := v.limiter.ReserveN(now, 1)
	if delay := r.DelayFrom(now); delay > 0 {
		r.CancelAt(now)
		return delay
	}
	return 0
}

// Sweep evicts the IPs idle for longer than IdleTTL.
func (l *RateLimiter) Sweep() {
	cutoff := l.opts.Now().Add(-l.opts.IdleTTL)
	l.mu.Lock()
	defer l.mu.Unlock()
	for ip, v := range l.visitors {
		if v.lastSeen.Before(cutoff) {
			delete(l.visitors, ip)
		}
	}
}

// Len returns how many IPs are tracked.
func (l *RateLimiter) Len() int {
	l.mu.Lock()
	defer l.mu.Unlock()
	return len(l.visitors)
}

// RunJanitor sweeps every interval until ctx is canceled.
func (l *RateLimiter) RunJanitor(ctx context.Context, interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			l.Sweep()
		}
	}
}
