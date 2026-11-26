package middleware_test

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/YaelBadillo/product-catalog/api/internal/httpapi/middleware"
)

// clock is a manual clock for the limiter.
type clock struct {
	mu  sync.Mutex
	now time.Time
}

func newClock() *clock { return &clock{now: time.Date(2026, 11, 25, 12, 0, 0, 0, time.UTC)} }

func (c *clock) Now() time.Time {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.now
}

func (c *clock) Advance(d time.Duration) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.now = c.now.Add(d)
}

type limited struct {
	engine *gin.Engine
	errs   []error
}

func newLimited(limiter *middleware.RateLimiter) *limited {
	l := &limited{engine: gin.New()}
	l.engine.Use(func(c *gin.Context) {
		c.Next()
		if last := c.Errors.Last(); last != nil {
			l.errs = append(l.errs, last.Err)
			c.Status(http.StatusTooManyRequests)
		}
	}, limiter.Middleware())
	l.engine.GET("/", func(c *gin.Context) { c.Status(http.StatusOK) })
	return l
}

func (l *limited) hit(t *testing.T, ip string) int {
	t.Helper()
	req := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/", http.NoBody)
	req.RemoteAddr = ip + ":1234"
	rec := httptest.NewRecorder()
	l.engine.ServeHTTP(rec, req)
	return rec.Code
}

func TestRateLimiterBurstThenLimit(t *testing.T) {
	clk := newClock()
	l := newLimited(middleware.NewRateLimiter(middleware.RateLimitOptions{Now: clk.Now}))

	for i := range 300 {
		require.Equal(t, http.StatusOK, l.hit(t, "192.0.2.1"), "request %d", i+1)
	}
	assert.Equal(t, http.StatusTooManyRequests, l.hit(t, "192.0.2.1"), "request 301")

	require.Len(t, l.errs, 1)
	var rl *middleware.RateLimitedError
	require.True(t, errors.As(l.errs[0], &rl))
	assert.Equal(t, 200*time.Millisecond, rl.RetryAfter, "one token every 200 ms")

	assert.Equal(t, http.StatusOK, l.hit(t, "192.0.2.2"), "another IP has its own bucket")
}

func TestRateLimiterRefillsAtFivePerSecond(t *testing.T) {
	clk := newClock()
	l := newLimited(middleware.NewRateLimiter(middleware.RateLimitOptions{Now: clk.Now}))
	for range 300 {
		l.hit(t, "192.0.2.1")
	}
	require.Equal(t, http.StatusTooManyRequests, l.hit(t, "192.0.2.1"))

	clk.Advance(time.Second)

	for i := range 5 {
		assert.Equal(t, http.StatusOK, l.hit(t, "192.0.2.1"), "refilled token %d", i+1)
	}
	assert.Equal(t, http.StatusTooManyRequests, l.hit(t, "192.0.2.1"))
}

func TestRateLimiterSweepEvictsIdleIPs(t *testing.T) {
	clk := newClock()
	limiter := middleware.NewRateLimiter(middleware.RateLimitOptions{Now: clk.Now})
	l := newLimited(limiter)
	l.hit(t, "192.0.2.1")
	clk.Advance(2 * time.Minute)
	l.hit(t, "192.0.2.2")
	require.Equal(t, 2, limiter.Len())

	clk.Advance(61 * time.Second)
	limiter.Sweep()

	assert.Equal(t, 1, limiter.Len(), "only the IP idle for more than 3 min is evicted")
}

func TestRateLimiterJanitorStopsWithItsContext(t *testing.T) {
	clk := newClock()
	limiter := middleware.NewRateLimiter(middleware.RateLimitOptions{Now: clk.Now})
	newLimited(limiter).hit(t, "192.0.2.1")
	clk.Advance(4 * time.Minute)
	ctx, cancel := context.WithCancel(t.Context())
	done := make(chan struct{})

	go func() {
		limiter.RunJanitor(ctx, time.Millisecond)
		close(done)
	}()

	assert.Eventually(t, func() bool { return limiter.Len() == 0 }, time.Second, time.Millisecond)
	cancel()
	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("the janitor did not stop")
	}
}
