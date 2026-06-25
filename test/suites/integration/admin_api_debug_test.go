package integration

import (
	"fmt"
	"net/http"
	"testing"

	adminapi "github.com/trebent/kerberos/test/integration/client/admin"
)

// startDebugSession is a helper that starts a debug session for the given backend and returns its ID.
func startDebugSession(t *testing.T, session, backend string) int {
	t.Helper()
	resp, err := adminClient.StartDebugSessionWithResponse(
		t.Context(),
		backend,
		adminapi.StartDebugSessionJSONRequestBody{},
		adminapi.RequestEditorFn(requestEditorSessionID(session)),
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

// --- StartDebugSession ---

// TestDebugStartSession verifies that a superuser can start a debug session and
// the response body contains the correct fields.
func TestDebugStartSession(t *testing.T) {
	superSession := superLogin(t)

	resp, err := adminClient.StartDebugSessionWithResponse(
		t.Context(),
		"echo",
		adminapi.StartDebugSessionJSONRequestBody{},
		adminapi.RequestEditorFn(requestEditorSessionID(superSession)),
	)
	checkErr(err, t)
	verifyStatusCode(resp.StatusCode(), http.StatusOK, t)

	session := resp.JSON200
	if session == nil {
		t.Fatal("expected non-nil debug session in response body")
	}
	if session.Id == 0 {
		t.Error("expected non-zero session ID")
	}
	if session.Backend != "echo" {
		t.Errorf("expected backend %q, got %q", "echo", session.Backend)
	}
	if session.StartedAt.IsZero() {
		t.Error("expected non-zero StartedAt")
	}
	if session.ExpiresAt.IsZero() {
		t.Error("expected non-zero ExpiresAt")
	}
	if session.StoppedAt != nil {
		t.Error("expected nil StoppedAt for a newly started session")
	}

	// Clean up.
	deleteResp, err := adminClient.DeleteDebugSessionWithResponse(
		t.Context(),
		"echo",
		session.Id,
		adminapi.RequestEditorFn(requestEditorSessionID(superSession)),
	)
	checkErr(err, t)
	verifyStatusCode(deleteResp.StatusCode(), http.StatusNoContent, t)
}

// TestDebugStartSessionConflict verifies that starting a second debug session for a
// backend that already has an active session returns 409 conflict.
func TestDebugStartSessionConflict(t *testing.T) {
	t.Parallel()
	superSession := superLogin(t)

	// Use a backend name unique to this test to avoid conflicts with parallel tests.
	const conflictBackend = "echo-conflict-test"

	sessionID := startDebugSession(t, superSession, conflictBackend)

	// Attempt to start a second session for the same backend.
	resp, err := adminClient.StartDebugSessionWithResponse(
		t.Context(),
		conflictBackend,
		adminapi.StartDebugSessionJSONRequestBody{},
		adminapi.RequestEditorFn(requestEditorSessionID(superSession)),
	)
	checkErr(err, t)
	verifyStatusCode(resp.StatusCode(), http.StatusConflict, t)
	verifyAdminAPIErrorResponse(resp.JSON409, t)

	// Clean up.
	deleteResp, err := adminClient.DeleteDebugSessionWithResponse(
		t.Context(),
		conflictBackend,
		sessionID,
		adminapi.RequestEditorFn(requestEditorSessionID(superSession)),
	)
	checkErr(err, t)
	verifyStatusCode(deleteResp.StatusCode(), http.StatusNoContent, t)
}

// --- ListDebugSessions ---

// TestDebugListSessionsEmpty verifies that listing debug sessions for a backend with no
// sessions returns 200 with an empty list.
func TestDebugListSessionsEmpty(t *testing.T) {
	t.Parallel()
	superSession := superLogin(t)

	// Use a backend name that no other test will use.
	const unusedBackend = "no-such-backend-for-list-test"

	resp, err := adminClient.ListDebugSessionsWithResponse(
		t.Context(),
		unusedBackend,
		adminapi.RequestEditorFn(requestEditorSessionID(superSession)),
	)
	checkErr(err, t)
	verifyStatusCode(resp.StatusCode(), http.StatusOK, t)

	if resp.JSON200 == nil {
		t.Fatal("expected non-nil sessions list")
	}
	if len(*resp.JSON200) != 0 {
		t.Errorf("expected empty sessions list, got %d sessions", len(*resp.JSON200))
	}
}

// TestDebugListSessionsContainsCreated verifies that a created session appears in the list.
func TestDebugListSessionsContainsCreated(t *testing.T) {
	superSession := superLogin(t)

	sessionID := startDebugSession(t, superSession, "echo")

	listResp, err := adminClient.ListDebugSessionsWithResponse(
		t.Context(),
		"echo",
		adminapi.RequestEditorFn(requestEditorSessionID(superSession)),
	)
	checkErr(err, t)
	verifyStatusCode(listResp.StatusCode(), http.StatusOK, t)

	found := false
	for _, s := range *listResp.JSON200 {
		if s.Id == sessionID {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("created session ID %d not found in list", sessionID)
	}

	// Clean up.
	deleteResp, err := adminClient.DeleteDebugSessionWithResponse(
		t.Context(),
		"echo",
		sessionID,
		adminapi.RequestEditorFn(requestEditorSessionID(superSession)),
	)
	checkErr(err, t)
	verifyStatusCode(deleteResp.StatusCode(), http.StatusNoContent, t)
}

// --- GetDebugSession ---

// TestDebugGetSession verifies that an existing session can be retrieved by ID.
func TestDebugGetSession(t *testing.T) {
	superSession := superLogin(t)

	sessionID := startDebugSession(t, superSession, "echo")

	getResp, err := adminClient.GetDebugSessionWithResponse(
		t.Context(),
		"echo",
		sessionID,
		adminapi.RequestEditorFn(requestEditorSessionID(superSession)),
	)
	checkErr(err, t)
	verifyStatusCode(getResp.StatusCode(), http.StatusOK, t)

	if getResp.JSON200 == nil {
		t.Fatal("expected non-nil debug session in response body")
	}
	if getResp.JSON200.Id != sessionID {
		t.Errorf("expected session ID %d, got %d", sessionID, getResp.JSON200.Id)
	}
	if getResp.JSON200.Backend != "echo" {
		t.Errorf("expected backend %q, got %q", "echo", getResp.JSON200.Backend)
	}

	// Clean up.
	deleteResp, err := adminClient.DeleteDebugSessionWithResponse(
		t.Context(),
		"echo",
		sessionID,
		adminapi.RequestEditorFn(requestEditorSessionID(superSession)),
	)
	checkErr(err, t)
	verifyStatusCode(deleteResp.StatusCode(), http.StatusNoContent, t)
}

// TestDebugGetSessionNotFound verifies that requesting a non-existent session returns 404.
func TestDebugGetSessionNotFound(t *testing.T) {
	t.Parallel()
	superSession := superLogin(t)

	resp, err := adminClient.GetDebugSessionWithResponse(
		t.Context(),
		"echo",
		999999999,
		adminapi.RequestEditorFn(requestEditorSessionID(superSession)),
	)
	checkErr(err, t)
	verifyStatusCode(resp.StatusCode(), http.StatusNotFound, t)
	verifyAdminAPIErrorResponse(resp.JSON404, t)
}

// --- ExtendDebugSession ---

// TestDebugExtendSession verifies that extending a session updates ExpiresAt.
func TestDebugExtendSession(t *testing.T) {
	superSession := superLogin(t)

	sessionID := startDebugSession(t, superSession, "echo")

	// Read original expiry.
	getResp, err := adminClient.GetDebugSessionWithResponse(
		t.Context(),
		"echo",
		sessionID,
		adminapi.RequestEditorFn(requestEditorSessionID(superSession)),
	)
	checkErr(err, t)
	verifyStatusCode(getResp.StatusCode(), http.StatusOK, t)
	originalExpiry := getResp.JSON200.ExpiresAt

	// Extend by 60 seconds.
	extResp, err := adminClient.ExtendDebugSessionWithResponse(
		t.Context(),
		"echo",
		sessionID,
		adminapi.ExtendDebugSessionJSONRequestBody{AdditionalDurationSeconds: 60},
		adminapi.RequestEditorFn(requestEditorSessionID(superSession)),
	)
	checkErr(err, t)
	verifyStatusCode(extResp.StatusCode(), http.StatusOK, t)

	if extResp.JSON200 == nil {
		t.Fatal("expected non-nil debug session in extend response body")
	}
	if !extResp.JSON200.ExpiresAt.After(originalExpiry) {
		t.Errorf("expected ExpiresAt to be after original %v, got %v", originalExpiry, extResp.JSON200.ExpiresAt)
	}

	// Clean up.
	deleteResp, err := adminClient.DeleteDebugSessionWithResponse(
		t.Context(),
		"echo",
		sessionID,
		adminapi.RequestEditorFn(requestEditorSessionID(superSession)),
	)
	checkErr(err, t)
	verifyStatusCode(deleteResp.StatusCode(), http.StatusNoContent, t)
}

// TestDebugExtendSessionNotFound verifies that extending a non-existent session returns 404.
func TestDebugExtendSessionNotFound(t *testing.T) {
	t.Parallel()
	superSession := superLogin(t)

	resp, err := adminClient.ExtendDebugSessionWithResponse(
		t.Context(),
		"echo",
		999999999,
		adminapi.ExtendDebugSessionJSONRequestBody{AdditionalDurationSeconds: 60},
		adminapi.RequestEditorFn(requestEditorSessionID(superSession)),
	)
	checkErr(err, t)
	verifyStatusCode(resp.StatusCode(), http.StatusNotFound, t)
	verifyAdminAPIErrorResponse(resp.JSON404, t)
}

// --- StopDebugSession ---

// TestDebugStopSession verifies that stopping an active session returns 204 and marks
// the session as stopped.
func TestDebugStopSession(t *testing.T) {
	superSession := superLogin(t)

	sessionID := startDebugSession(t, superSession, "echo")

	stopResp, err := adminClient.StopDebugSessionWithResponse(
		t.Context(),
		"echo",
		sessionID,
		adminapi.RequestEditorFn(requestEditorSessionID(superSession)),
	)
	checkErr(err, t)
	verifyStatusCode(stopResp.StatusCode(), http.StatusNoContent, t)

	// Verify StoppedAt is set.
	getResp, err := adminClient.GetDebugSessionWithResponse(
		t.Context(),
		"echo",
		sessionID,
		adminapi.RequestEditorFn(requestEditorSessionID(superSession)),
	)
	checkErr(err, t)
	verifyStatusCode(getResp.StatusCode(), http.StatusOK, t)
	if getResp.JSON200.StoppedAt == nil {
		t.Error("expected non-nil StoppedAt after stopping the session")
	}

	// Clean up.
	deleteResp, err := adminClient.DeleteDebugSessionWithResponse(
		t.Context(),
		"echo",
		sessionID,
		adminapi.RequestEditorFn(requestEditorSessionID(superSession)),
	)
	checkErr(err, t)
	verifyStatusCode(deleteResp.StatusCode(), http.StatusNoContent, t)
}

// TestDebugStopSessionNotFound verifies that stopping a non-existent session returns 404.
func TestDebugStopSessionNotFound(t *testing.T) {
	t.Parallel()
	superSession := superLogin(t)

	resp, err := adminClient.StopDebugSessionWithResponse(
		t.Context(),
		"echo",
		999999999,
		adminapi.RequestEditorFn(requestEditorSessionID(superSession)),
	)
	checkErr(err, t)
	verifyStatusCode(resp.StatusCode(), http.StatusNotFound, t)
	verifyAdminAPIErrorResponse(resp.JSON404, t)
}

// --- DeleteDebugSession ---

// TestDebugDeleteSession verifies that deleting a session returns 204 and the session
// is no longer retrievable.
func TestDebugDeleteSession(t *testing.T) {
	superSession := superLogin(t)

	sessionID := startDebugSession(t, superSession, "echo")

	deleteResp, err := adminClient.DeleteDebugSessionWithResponse(
		t.Context(),
		"echo",
		sessionID,
		adminapi.RequestEditorFn(requestEditorSessionID(superSession)),
	)
	checkErr(err, t)
	verifyStatusCode(deleteResp.StatusCode(), http.StatusNoContent, t)

	// Verify the session is gone.
	getResp, err := adminClient.GetDebugSessionWithResponse(
		t.Context(),
		"echo",
		sessionID,
		adminapi.RequestEditorFn(requestEditorSessionID(superSession)),
	)
	checkErr(err, t)
	verifyStatusCode(getResp.StatusCode(), http.StatusNotFound, t)
}

// TestDebugDeleteSessionNotFound verifies that deleting a non-existent session returns 404.
func TestDebugDeleteSessionNotFound(t *testing.T) {
	t.Parallel()
	superSession := superLogin(t)

	resp, err := adminClient.DeleteDebugSessionWithResponse(
		t.Context(),
		"echo",
		999999999,
		adminapi.RequestEditorFn(requestEditorSessionID(superSession)),
	)
	checkErr(err, t)
	verifyStatusCode(resp.StatusCode(), http.StatusNotFound, t)
	verifyAdminAPIErrorResponse(resp.JSON404, t)
}

// --- ListDebugSessionCalls & GetDebugSessionCall ---

// TestDebugListSessionCallsWithTransitions verifies that after a gateway request is made
// during an active debug session, the call is recorded and flow transitions are populated
// when includeTransitions=true.
func TestDebugListSessionCallsWithTransitions(t *testing.T) {
	superSession := superLogin(t)

	sessionID := startDebugSession(t, superSession, "echo")
	defer func() {
		deleteResp, err := adminClient.DeleteDebugSessionWithResponse(
			t.Context(),
			"echo",
			sessionID,
			adminapi.RequestEditorFn(requestEditorSessionID(superSession)),
		)
		checkErr(err, t)
		verifyStatusCode(deleteResp.StatusCode(), http.StatusNoContent, t)
	}()

	makeGatewayRequest(t, "echo", "/hi")

	listResp, err := adminClient.ListDebugSessionCallsWithResponse(
		t.Context(),
		"echo",
		sessionID,
		&adminapi.ListDebugSessionCallsParams{IncludeTransitions: true},
		adminapi.RequestEditorFn(requestEditorSessionID(superSession)),
	)
	checkErr(err, t)
	verifyStatusCode(listResp.StatusCode(), http.StatusOK, t)

	if listResp.JSON200 == nil {
		t.Fatal("expected non-nil calls list")
	}
	if len(*listResp.JSON200) == 0 {
		t.Fatal("expected at least one call to be recorded")
	}
	call := (*listResp.JSON200)[0]
	if len(call.FlowTransitions) == 0 {
		t.Error("expected non-empty FlowTransitions when includeTransitions=true")
	}
}

// TestDebugListSessionCallsWithoutTransitions verifies that when includeTransitions=false,
// FlowTransitions are not included in the response.
func TestDebugListSessionCallsWithoutTransitions(t *testing.T) {
	superSession := superLogin(t)

	sessionID := startDebugSession(t, superSession, "echo")
	defer func() {
		deleteResp, err := adminClient.DeleteDebugSessionWithResponse(
			t.Context(),
			"echo",
			sessionID,
			adminapi.RequestEditorFn(requestEditorSessionID(superSession)),
		)
		checkErr(err, t)
		verifyStatusCode(deleteResp.StatusCode(), http.StatusNoContent, t)
	}()

	makeGatewayRequest(t, "echo", "/hi")

	listResp, err := adminClient.ListDebugSessionCallsWithResponse(
		t.Context(),
		"echo",
		sessionID,
		&adminapi.ListDebugSessionCallsParams{IncludeTransitions: false},
		adminapi.RequestEditorFn(requestEditorSessionID(superSession)),
	)
	checkErr(err, t)
	verifyStatusCode(listResp.StatusCode(), http.StatusOK, t)

	if listResp.JSON200 == nil {
		t.Fatal("expected non-nil calls list")
	}
	if len(*listResp.JSON200) == 0 {
		t.Fatal("expected at least one call to be recorded")
	}
	for _, call := range *listResp.JSON200 {
		if len(call.FlowTransitions) != 0 {
			t.Error("expected empty FlowTransitions when includeTransitions=false")
		}
	}
}

// TestDebugGetSessionCall verifies that a specific recorded call can be retrieved by ID.
func TestDebugGetSessionCall(t *testing.T) {
	superSession := superLogin(t)

	sessionID := startDebugSession(t, superSession, "echo")
	defer func() {
		deleteResp, err := adminClient.DeleteDebugSessionWithResponse(
			t.Context(),
			"echo",
			sessionID,
			adminapi.RequestEditorFn(requestEditorSessionID(superSession)),
		)
		checkErr(err, t)
		verifyStatusCode(deleteResp.StatusCode(), http.StatusNoContent, t)
	}()

	makeGatewayRequest(t, "echo", "/hi")

	listResp, err := adminClient.ListDebugSessionCallsWithResponse(
		t.Context(),
		"echo",
		sessionID,
		&adminapi.ListDebugSessionCallsParams{IncludeTransitions: false},
		adminapi.RequestEditorFn(requestEditorSessionID(superSession)),
	)
	checkErr(err, t)
	verifyStatusCode(listResp.StatusCode(), http.StatusOK, t)

	if listResp.JSON200 == nil || len(*listResp.JSON200) == 0 {
		t.Fatal("expected at least one recorded call")
	}
	callID := (*listResp.JSON200)[0].Id

	getCallResp, err := adminClient.GetDebugSessionCallWithResponse(
		t.Context(),
		"echo",
		sessionID,
		callID,
		adminapi.RequestEditorFn(requestEditorSessionID(superSession)),
	)
	checkErr(err, t)
	verifyStatusCode(getCallResp.StatusCode(), http.StatusOK, t)

	if getCallResp.JSON200 == nil {
		t.Fatal("expected non-nil call in response body")
	}
	if getCallResp.JSON200.Id != callID {
		t.Errorf("expected call ID %d, got %d", callID, getCallResp.JSON200.Id)
	}
	if getCallResp.JSON200.Method == "" {
		t.Error("expected non-empty Method")
	}
	if getCallResp.JSON200.Url == "" {
		t.Error("expected non-empty Url")
	}
}

// TestDebugGetSessionCallNotFound verifies that requesting a non-existent call returns 404.
func TestDebugGetSessionCallNotFound(t *testing.T) {
	superSession := superLogin(t)

	sessionID := startDebugSession(t, superSession, "echo")
	defer func() {
		deleteResp, err := adminClient.DeleteDebugSessionWithResponse(
			t.Context(),
			"echo",
			sessionID,
			adminapi.RequestEditorFn(requestEditorSessionID(superSession)),
		)
		checkErr(err, t)
		verifyStatusCode(deleteResp.StatusCode(), http.StatusNoContent, t)
	}()

	resp, err := adminClient.GetDebugSessionCallWithResponse(
		t.Context(),
		"echo",
		sessionID,
		999999999,
		adminapi.RequestEditorFn(requestEditorSessionID(superSession)),
	)
	checkErr(err, t)
	verifyStatusCode(resp.StatusCode(), http.StatusNotFound, t)
	verifyAdminAPIErrorResponse(resp.JSON404, t)
}

// --- Full lifecycle ---

// TestDebugFullFlow exercises the complete debug session lifecycle end-to-end:
// start → get → hit gateway (records a call) → list calls → get call → stop → delete.
func TestDebugFullFlow(t *testing.T) {
	superSession := superLogin(t)

	// Start.
	startResp, err := adminClient.StartDebugSessionWithResponse(
		t.Context(),
		"echo",
		adminapi.StartDebugSessionJSONRequestBody{},
		adminapi.RequestEditorFn(requestEditorSessionID(superSession)),
	)
	checkErr(err, t)
	verifyStatusCode(startResp.StatusCode(), http.StatusOK, t)
	sessionID := startResp.JSON200.Id

	// Get.
	getResp, err := adminClient.GetDebugSessionWithResponse(
		t.Context(),
		"echo",
		sessionID,
		adminapi.RequestEditorFn(requestEditorSessionID(superSession)),
	)
	checkErr(err, t)
	verifyStatusCode(getResp.StatusCode(), http.StatusOK, t)
	matches(getResp.JSON200.Id, sessionID, t)

	// Make a gateway request so a call gets recorded.
	makeGatewayRequest(t, "echo", "/flow-test")

	// List calls with transitions.
	listCallsResp, err := adminClient.ListDebugSessionCallsWithResponse(
		t.Context(),
		"echo",
		sessionID,
		&adminapi.ListDebugSessionCallsParams{IncludeTransitions: true},
		adminapi.RequestEditorFn(requestEditorSessionID(superSession)),
	)
	checkErr(err, t)
	verifyStatusCode(listCallsResp.StatusCode(), http.StatusOK, t)
	if len(*listCallsResp.JSON200) == 0 {
		t.Fatal("expected at least one recorded call after gateway request")
	}
	callID := (*listCallsResp.JSON200)[0].Id

	// Get specific call.
	getCallResp, err := adminClient.GetDebugSessionCallWithResponse(
		t.Context(),
		"echo",
		sessionID,
		callID,
		adminapi.RequestEditorFn(requestEditorSessionID(superSession)),
	)
	checkErr(err, t)
	verifyStatusCode(getCallResp.StatusCode(), http.StatusOK, t)
	matches(getCallResp.JSON200.Id, callID, t)

	// Stop.
	stopResp, err := adminClient.StopDebugSessionWithResponse(
		t.Context(),
		"echo",
		sessionID,
		adminapi.RequestEditorFn(requestEditorSessionID(superSession)),
	)
	checkErr(err, t)
	verifyStatusCode(stopResp.StatusCode(), http.StatusNoContent, t)

	// Delete.
	deleteResp, err := adminClient.DeleteDebugSessionWithResponse(
		t.Context(),
		"echo",
		sessionID,
		adminapi.RequestEditorFn(requestEditorSessionID(superSession)),
	)
	checkErr(err, t)
	verifyStatusCode(deleteResp.StatusCode(), http.StatusNoContent, t)
}
