package coderd_test

import (
	"context"
	"net/http"

	"github.com/stretchr/testify/require"

	"github.com/coder/coder/codersdk"
	"github.com/coder/coder/testutil"
)

// Issue: https://github.com/coder/coder/issues/5249
// While running tests in parallel, the web server seems to be overloaded and responds with HTTP 502.
// require.Eventually expects correct HTTP responses.
func doWithRetries(t require.TestingT, client *codersdk.Client, req *http.Request) (*http.Response, error) {
	var resp *http.Response
	var err error
	require.Eventually(t, func() bool {
		resp, err = client.HTTPClient.Do(req)
		return resp.StatusCode != http.StatusBadGateway
	}, testutil.WaitLong, testutil.IntervalFast)
	return resp, err
}

func requestWithRetries(t require.TestingT, client *codersdk.Client, ctx context.Context, method, path string, body interface{}, opts ...codersdk.RequestOption) (*http.Response, error) {
	var resp *http.Response
	var err error
	require.Eventually(t, func() bool {
		resp, err = client.Request(ctx, method, path, body, opts...)
		return resp == nil || resp.StatusCode != http.StatusBadGateway
	}, testutil.WaitLong, testutil.IntervalFast)
	return resp, err
}
