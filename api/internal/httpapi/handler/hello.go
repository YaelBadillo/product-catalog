// Package handler contains the HTTP handlers and their DTOs.
package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// HelloResponse is the body returned by Hello.
type HelloResponse struct {
	Message string `json:"message"`
}

// Hello responds with a greeting. It only exists to verify the setup.
func Hello(c *gin.Context) {
	c.JSON(http.StatusOK, HelloResponse{Message: "Hello, world!"})
}
