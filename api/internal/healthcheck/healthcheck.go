// Package healthcheck probes the API's liveness endpoint. It backs the
// binary's `healthcheck` subcommand, used by container health checks where
// the distroless image has no curl.
package healthcheck

import (
	"context"
	"fmt"
	"net/http"
)

// Check sends GET url and returns an error unless the response is 200 OK.
// ctx bounds the whole request.
func Check(ctx context.Context, url string) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, http.NoBody)
	if err != nil {
		return fmt.Errorf("build request: %w", err)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return fmt.Errorf("request %s: %w", url, err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("request %s: unexpected status %d", url, resp.StatusCode)
	}
	return nil
}
