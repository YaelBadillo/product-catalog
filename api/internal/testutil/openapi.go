// Package testutil holds helpers shared by tests.
package testutil

import (
	"bytes"
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"runtime"
	"sync"

	"github.com/getkin/kin-openapi/openapi3"
	"github.com/getkin/kin-openapi/openapi3filter"
	"github.com/getkin/kin-openapi/routers"
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
	rewindBody(t, req)
	reqInput := findRoute(t, req)
	require.NoError(t, openapi3filter.ValidateRequest(ctx, reqInput), "request does not match openapi.yaml")
	validateResponse(ctx, t, reqInput, rec)
}

// AssertResponseMatchesContract fails the test unless the response recorded
// in rec conforms to openapi.yaml for the route of req. Unlike
// AssertMatchesContract it does not validate the request, so tests can send
// intentionally invalid requests and still check the response they get.
func AssertResponseMatchesContract(t TestingT, req *http.Request, rec *httptest.ResponseRecorder) {
	t.Helper()
	validateResponse(req.Context(), t, findRoute(t, req), rec)
}

// rewindBody restores a body the handler already read, when the request can
// produce it again (req.GetBody).
func rewindBody(t TestingT, req *http.Request) {
	t.Helper()
	if req.GetBody == nil {
		return
	}
	body, err := req.GetBody()
	require.NoError(t, err, "rewind request body")
	req.Body = body
}

func findRoute(t TestingT, req *http.Request) *openapi3filter.RequestValidationInput {
	t.Helper()
	router := contractRouter(t)
	route, pathParams, err := router.FindRoute(req)
	require.NoError(t, err, "%s %s is not in openapi.yaml", req.Method, req.URL.Path)
	return &openapi3filter.RequestValidationInput{
		Request:    req,
		PathParams: pathParams,
		Route:      route,
		Options:    &openapi3filter.Options{AuthenticationFunc: openapi3filter.NoopAuthenticationFunc},
	}
}

func validateResponse(ctx context.Context, t TestingT, reqInput *openapi3filter.RequestValidationInput, rec *httptest.ResponseRecorder) {
	t.Helper()
	err := openapi3filter.ValidateResponse(ctx, &openapi3filter.ResponseValidationInput{
		RequestValidationInput: reqInput,
		Status:                 rec.Code,
		Header:                 rec.Header(),
		Body:                   io.NopCloser(bytes.NewReader(rec.Body.Bytes())),
		Options:                &openapi3filter.Options{IncludeResponseStatus: true},
	})
	require.NoError(t, err, "response does not match openapi.yaml")
}

var errLocateSource = errors.New("locate testutil source file")

var contract struct {
	once   sync.Once
	router routers.Router
	err    error
}

// contractRouter loads and validates openapi.yaml once per test binary.
func contractRouter(t TestingT) routers.Router {
	t.Helper()
	contract.once.Do(func() {
		contract.router, contract.err = loadContract()
	})
	require.NoError(t, contract.err, "load openapi.yaml")
	return contract.router
}

// documentTypes are bodies the contract describes as a plain string (the
// /docs page and the contract itself), so they are validated as text.
var documentTypes = []string{"text/html", "application/yaml"}

func loadContract() (routers.Router, error) {
	for _, contentType := range documentTypes {
		openapi3filter.RegisterBodyDecoder(contentType, openapi3filter.PlainBodyDecoder)
	}
	path, err := contractPath()
	if err != nil {
		return nil, err
	}
	loader := openapi3.NewLoader()
	doc, err := loader.LoadFromFile(path)
	if err != nil {
		return nil, err
	}
	if err := doc.Validate(loader.Context); err != nil {
		return nil, err
	}
	return gorillamux.NewRouter(doc)
}

// ContractPath returns the path of openapi.yaml at the repository root.
func ContractPath() (string, error) { return contractPath() }

func contractPath() (string, error) {
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		return "", errLocateSource
	}
	return filepath.Join(filepath.Dir(file), "..", "..", "..", "openapi.yaml"), nil
}
