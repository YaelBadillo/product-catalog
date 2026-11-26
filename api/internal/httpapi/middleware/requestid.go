// Package middleware contains the HTTP middleware shared by every route.
package middleware

import (
	"context"
	"crypto/rand"
	"regexp"

	"github.com/gin-gonic/gin"
)

// RequestIDHeader carries the request ID in requests and responses.
const RequestIDHeader = "X-Request-ID"

// validRequestID bounds incoming IDs so they cannot inject content into logs.
var validRequestID = regexp.MustCompile(`^[A-Za-z0-9-]{1,64}$`)

type requestIDKey struct{}

// RequestID reuses a valid incoming X-Request-ID or generates a new one,
// echoes it in the response and stores it in the request context.
func RequestID() gin.HandlerFunc {
	return func(c *gin.Context) {
		id := c.GetHeader(RequestIDHeader)
		if !validRequestID.MatchString(id) {
			id = rand.Text()
		}
		c.Header(RequestIDHeader, id)
		c.Request = c.Request.WithContext(context.WithValue(c.Request.Context(), requestIDKey{}, id))
		c.Next()
	}
}

// RequestIDFrom returns the request ID stored in ctx, or "" if there is none.
func RequestIDFrom(ctx context.Context) string {
	id, _ := ctx.Value(requestIDKey{}).(string)
	return id
}
