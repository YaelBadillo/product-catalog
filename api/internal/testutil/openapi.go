// Package testutil holds helpers shared by tests.
package testutil

import (
	"bytes"
	"io"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"runtime"

	"github.com/getkin/kin-openapi/openapi3"
	"github.com/getkin/kin-openapi/openapi3filter"
	"github.com/getkin/kin-openapi/routers/gorillamux"
	"github.com/stretchr/testify/require"
)

// TestingT is the subset of *testing.T used by the assertions in this package.
type TestingT interface {
	require.TestingT
	Helper()
}

// AssertMatchesContract fails the test unless req and the response recorded
// in rec conform to the repository's openapi.yaml.
func AssertMatchesContract(t TestingT, req *http.Request, rec *httptest.ResponseRecorder) {
	t.Helper()
	ctx := req.Context()

	loader := openapi3.NewLoader()
	doc, err := loader.LoadFromFile(contractPath(t))
	require.NoError(t, err, "load openapi.yaml")
	require.NoError(t, doc.Validate(loader.Context), "validate openapi.yaml")

	router, err := gorillamux.NewRouter(doc)
	require.NoError(t, err, "build openapi router")
	route, pathParams, err := router.FindRoute(req)
	require.NoError(t, err, "%s %s is not in openapi.yaml", req.Method, req.URL.Path)

	reqInput := &openapi3filter.RequestValidationInput{
		Request:    req,
		PathParams: pathParams,
		Route:      route,
		Options:    &openapi3filter.Options{AuthenticationFunc: openapi3filter.NoopAuthenticationFunc},
	}
	require.NoError(t, openapi3filter.ValidateRequest(ctx, reqInput), "request does not match openapi.yaml")

	err = openapi3filter.ValidateResponse(ctx, &openapi3filter.ResponseValidationInput{
		RequestValidationInput: reqInput,
		Status:                 rec.Code,
		Header:                 rec.Header(),
		Body:                   io.NopCloser(bytes.NewReader(rec.Body.Bytes())),
		Options:                &openapi3filter.Options{IncludeResponseStatus: true},
	})
	require.NoError(t, err, "response does not match openapi.yaml")
}

// contractPath returns the path of openapi.yaml at the repository root.
func contractPath(t TestingT) string {
	t.Helper()
	_, file, _, ok := runtime.Caller(0)
	require.True(t, ok, "locate testutil source file")
	return filepath.Join(filepath.Dir(file), "..", "..", "..", "openapi.yaml")
}
