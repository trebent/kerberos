package lib

import (
	"fmt"
	"net/http"
	"testing"

	adminapi "github.com/trebent/kerberos/test/client/admin"
)

// StartDebugSession starts a debug session for the given backend and returns its ID.
func StartDebugSession(t *testing.T, requestEditor RequestEditorFn, backend string) int {
	t.Helper()
	resp, err := AdminClient.StartDebugSessionWithResponse(
		t.Context(),
		backend,
		adminapi.StartDebugSessionJSONRequestBody{},
		adminapi.RequestEditorFn(requestEditor),
	)
	CheckErr(err, t)
	VerifyStatusCode(resp.StatusCode(), http.StatusOK, t)
	return resp.JSON200.Id
}

// MakeGatewayRequest sends a GET request through the gateway to the given backend path.
func MakeGatewayRequest(t *testing.T, backend, path string) {
	t.Helper()
	url := fmt.Sprintf("http://localhost:%d/gw/backend/%s%s", GetPort(), backend, path)
	resp := Get(url, t)
	defer resp.Body.Close()
}
