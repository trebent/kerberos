package integration

import (
	"fmt"
	"net/http"
	"testing"

	adminapi "github.com/trebent/kerberos/test/integration/client/admin"
)

// startDebugSession is a helper that starts a debug session for the given backend and returns its ID.
func startDebugSession(t *testing.T, requestEditor RequestEditorFn, backend string) int {
	t.Helper()
	resp, err := adminClient.StartDebugSessionWithResponse(
		t.Context(),
		backend,
		adminapi.StartDebugSessionJSONRequestBody{},
		adminapi.RequestEditorFn(requestEditor),
	)
	checkErr(err, t)
	verifyStatusCode(resp.StatusCode(), http.StatusOK, t)
	return resp.JSON200.Id
}

// makeGatewayRequest sends a GET request through the gateway to the given backend path.
func makeGatewayRequest(t *testing.T, backend, path string) {
	t.Helper()
	url := fmt.Sprintf("http://localhost:%d/gw/backend/%s%s", getPort(), backend, path)
	resp := get(url, t)
	defer resp.Body.Close()
}
