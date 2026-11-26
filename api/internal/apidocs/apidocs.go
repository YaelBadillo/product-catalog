// Package apidocs serves the browsable API documentation: Swagger UI at /docs
// over the embedded contract at /docs/openapi.yaml.
package apidocs

import (
	"crypto/sha256"
	_ "embed" // embeds the contract and the page
	"encoding/base64"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

// openAPI is a copy of the root openapi.yaml (make sync-openapi): go:embed
// cannot reach files outside the module.
//
//go:embed openapi.yaml
var openAPI []byte

//go:embed index.html
var page string

// initScript starts Swagger UI; the CSP allows exactly this inline script.
const initScript = `window.ui = SwaggerUIBundle({ url: "/docs/openapi.yaml", dom_id: "#swagger-ui" });`

var (
	html = strings.Replace(page, "{{INIT_SCRIPT}}", initScript, 1)
	csp  = strings.Join([]string{
		"default-src 'none'",
		"script-src https://cdn.jsdelivr.net 'sha256-" + scriptHash(initScript) + "'",
		// Swagger UI sets inline style attributes.
		"style-src https://cdn.jsdelivr.net 'unsafe-inline'",
		"img-src 'self' data: https://cdn.jsdelivr.net",
		"connect-src 'self'",
		"frame-ancestors 'none'",
	}, "; ")
)

func scriptHash(script string) string {
	sum := sha256.Sum256([]byte(script))
	return base64.StdEncoding.EncodeToString(sum[:])
}

// Register adds /docs and /docs/openapi.yaml to r.
func Register(r gin.IRoutes) {
	r.GET("/docs", Page)
	r.GET("/docs/openapi.yaml", Contract)
}

// Page serves Swagger UI with its own Content-Security-Policy.
func Page(c *gin.Context) {
	c.Header("Content-Security-Policy", csp)
	c.Data(http.StatusOK, "text/html; charset=utf-8", []byte(html))
}

// Contract serves the embedded openapi.yaml.
func Contract(c *gin.Context) {
	c.Data(http.StatusOK, "application/yaml", openAPI)
}
