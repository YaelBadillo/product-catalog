// Package httpapi wires the HTTP router.
package httpapi

import (
	"github.com/gin-gonic/gin"

	"github.com/YaelBadillo/product-catalog/api/internal/httpapi/handler"
)

// NewRouter builds the Gin engine with every route registered.
func NewRouter() *gin.Engine {
	router := gin.New()
	router.Use(gin.Recovery())

	v1 := router.Group("/api/v1")
	v1.GET("/hello", handler.Hello)

	return router
}
