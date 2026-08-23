package integration

import (
	lib "github.com/trebent/kerberos/test/lib"
	"net/http"
	"testing"

	adminapi "github.com/trebent/kerberos/test/client/admin"
)

// --- StartDebugSession ---

// TestAdminDebugStartSession verifies that a superuser can start a debug session and
// the response body contains the correct fields.
func TestAdminDebugStartSession(t *testing.T) {
	superRequestEditor := lib.SuperLogin(t)

	resp, err := lib.AdminClient.StartDebugSessionWithResponse(
		t.Context(),
		"echo",
		adminapi.StartDebugSessionJSONRequestBody{},
		adminapi.RequestEditorFn(superRequestEditor),
	)
	lib.CheckErr(err, t)
	lib.VerifyStatusCode(resp.StatusCode(), http.StatusOK, t)

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
	deleteResp, err := lib.AdminClient.DeleteDebugSessionWithResponse(
		t.Context(),
		"echo",
		session.Id,
		adminapi.RequestEditorFn(superRequestEditor),
	)
	lib.CheckErr(err, t)
	lib.VerifyStatusCode(deleteResp.StatusCode(), http.StatusNoContent, t)
}

// TestAdminDebugStartSessionConflict verifies that starting a second debug session for a
// backend that already has an active session returns 409 conflict.
func TestAdminDebugStartSessionConflict(t *testing.T) {
	t.Parallel()
	superRequestEditor := lib.SuperLogin(t)

	// Use a backend name unique to this test to avoid conflicts with parallel tests.
	const conflictBackend = "echo-conflict-test"

	sessionID := lib.StartDebugSession(t, superRequestEditor, conflictBackend)

	// Attempt to start a second session for the same backend.
	resp, err := lib.AdminClient.StartDebugSessionWithResponse(
		t.Context(),
		conflictBackend,
		adminapi.StartDebugSessionJSONRequestBody{},
		adminapi.RequestEditorFn(superRequestEditor),
	)
	lib.CheckErr(err, t)
	lib.VerifyStatusCode(resp.StatusCode(), http.StatusConflict, t)
	lib.VerifyAdminAPIErrorResponse(resp.JSON409, t)

	// Clean up.
	deleteResp, err := lib.AdminClient.DeleteDebugSessionWithResponse(
		t.Context(),
		conflictBackend,
		sessionID,
		adminapi.RequestEditorFn(superRequestEditor),
	)
	lib.CheckErr(err, t)
	lib.VerifyStatusCode(deleteResp.StatusCode(), http.StatusNoContent, t)
}

// --- ListDebugSessions ---

// TestAdminDebugListSessionsEmpty verifies that listing debug sessions for a backend with no
// sessions returns 200 with an empty list.
func TestAdminDebugListSessionsEmpty(t *testing.T) {
	t.Parallel()
	superRequestEditor := lib.SuperLogin(t)

	// Use a backend name that no other test will use.
	const unusedBackend = "no-such-backend-for-list-test"

	resp, err := lib.AdminClient.ListDebugSessionsWithResponse(
		t.Context(),
		unusedBackend,
		adminapi.RequestEditorFn(superRequestEditor),
	)
	lib.CheckErr(err, t)
	lib.VerifyStatusCode(resp.StatusCode(), http.StatusOK, t)

	if resp.JSON200 == nil {
		t.Fatal("expected non-nil sessions list")
	}
	if len(*resp.JSON200) != 0 {
		t.Errorf("expected empty sessions list, got %d sessions", len(*resp.JSON200))
	}
}

// TestAdminDebugListSessionsContainsCreated verifies that a created session appears in the list.
func TestAdminDebugListSessionsContainsCreated(t *testing.T) {
	superRequestEditor := lib.SuperLogin(t)

	sessionID := lib.StartDebugSession(t, superRequestEditor, "echo")

	listResp, err := lib.AdminClient.ListDebugSessionsWithResponse(
		t.Context(),
		"echo",
		adminapi.RequestEditorFn(superRequestEditor),
	)
	lib.CheckErr(err, t)
	lib.VerifyStatusCode(listResp.StatusCode(), http.StatusOK, t)

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
	deleteResp, err := lib.AdminClient.DeleteDebugSessionWithResponse(
		t.Context(),
		"echo",
		sessionID,
		adminapi.RequestEditorFn(superRequestEditor),
	)
	lib.CheckErr(err, t)
	lib.VerifyStatusCode(deleteResp.StatusCode(), http.StatusNoContent, t)
}

// --- GetDebugSession ---

// TestAdminDebugGetSession verifies that an existing session can be retrieved by ID.
func TestAdminDebugGetSession(t *testing.T) {
	superRequestEditor := lib.SuperLogin(t)

	sessionID := lib.StartDebugSession(t, superRequestEditor, "echo")

	getResp, err := lib.AdminClient.GetDebugSessionWithResponse(
		t.Context(),
		"echo",
		sessionID,
		adminapi.RequestEditorFn(superRequestEditor),
	)
	lib.CheckErr(err, t)
	lib.VerifyStatusCode(getResp.StatusCode(), http.StatusOK, t)

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
	deleteResp, err := lib.AdminClient.DeleteDebugSessionWithResponse(
		t.Context(),
		"echo",
		sessionID,
		adminapi.RequestEditorFn(superRequestEditor),
	)
	lib.CheckErr(err, t)
	lib.VerifyStatusCode(deleteResp.StatusCode(), http.StatusNoContent, t)
}

// TestAdminDebugGetSessionNotFound verifies that requesting a non-existent session returns 404.
func TestAdminDebugGetSessionNotFound(t *testing.T) {
	t.Parallel()
	superRequestEditor := lib.SuperLogin(t)

	resp, err := lib.AdminClient.GetDebugSessionWithResponse(
		t.Context(),
		"echo",
		999999999,
		adminapi.RequestEditorFn(superRequestEditor),
	)
	lib.CheckErr(err, t)
	lib.VerifyStatusCode(resp.StatusCode(), http.StatusNotFound, t)
	lib.VerifyAdminAPIErrorResponse(resp.JSON404, t)
}

// --- ExtendDebugSession ---

// TestAdminDebugExtendSession verifies that extending a session updates ExpiresAt.
func TestAdminDebugExtendSession(t *testing.T) {
	superRequestEditor := lib.SuperLogin(t)

	sessionID := lib.StartDebugSession(t, superRequestEditor, "echo")

	// Read original expiry.
	getResp, err := lib.AdminClient.GetDebugSessionWithResponse(
		t.Context(),
		"echo",
		sessionID,
		adminapi.RequestEditorFn(superRequestEditor),
	)
	lib.CheckErr(err, t)
	lib.VerifyStatusCode(getResp.StatusCode(), http.StatusOK, t)
	originalExpiry := getResp.JSON200.ExpiresAt

	// Extend by 60 seconds.
	extResp, err := lib.AdminClient.ExtendDebugSessionWithResponse(
		t.Context(),
		"echo",
		sessionID,
		adminapi.ExtendDebugSessionJSONRequestBody{AdditionalDurationSeconds: 60},
		adminapi.RequestEditorFn(superRequestEditor),
	)
	lib.CheckErr(err, t)
	lib.VerifyStatusCode(extResp.StatusCode(), http.StatusOK, t)

	if extResp.JSON200 == nil {
		t.Fatal("expected non-nil debug session in extend response body")
	}
	if !extResp.JSON200.ExpiresAt.After(originalExpiry) {
		t.Errorf("expected ExpiresAt to be after original %v, got %v", originalExpiry, extResp.JSON200.ExpiresAt)
	}

	// Clean up.
	deleteResp, err := lib.AdminClient.DeleteDebugSessionWithResponse(
		t.Context(),
		"echo",
		sessionID,
		adminapi.RequestEditorFn(superRequestEditor),
	)
	lib.CheckErr(err, t)
	lib.VerifyStatusCode(deleteResp.StatusCode(), http.StatusNoContent, t)
}

// TestAdminDebugExtendSessionNotFound verifies that extending a non-existent session returns 404.
func TestAdminDebugExtendSessionNotFound(t *testing.T) {
	t.Parallel()
	superRequestEditor := lib.SuperLogin(t)

	resp, err := lib.AdminClient.ExtendDebugSessionWithResponse(
		t.Context(),
		"echo",
		999999999,
		adminapi.ExtendDebugSessionJSONRequestBody{AdditionalDurationSeconds: 60},
		adminapi.RequestEditorFn(superRequestEditor),
	)
	lib.CheckErr(err, t)
	lib.VerifyStatusCode(resp.StatusCode(), http.StatusNotFound, t)
	lib.VerifyAdminAPIErrorResponse(resp.JSON404, t)
}

// --- StopDebugSession ---

// TestAdminDebugStopSession verifies that stopping an active session returns 204 and marks
// the session as stopped.
func TestAdminDebugStopSession(t *testing.T) {
	superRequestEditor := lib.SuperLogin(t)

	sessionID := lib.StartDebugSession(t, superRequestEditor, "echo")

	stopResp, err := lib.AdminClient.StopDebugSessionWithResponse(
		t.Context(),
		"echo",
		sessionID,
		adminapi.RequestEditorFn(superRequestEditor),
	)
	lib.CheckErr(err, t)
	lib.VerifyStatusCode(stopResp.StatusCode(), http.StatusNoContent, t)

	// Verify StoppedAt is set.
	getResp, err := lib.AdminClient.GetDebugSessionWithResponse(
		t.Context(),
		"echo",
		sessionID,
		adminapi.RequestEditorFn(superRequestEditor),
	)
	lib.CheckErr(err, t)
	lib.VerifyStatusCode(getResp.StatusCode(), http.StatusOK, t)
	if getResp.JSON200.StoppedAt == nil {
		t.Error("expected non-nil StoppedAt after stopping the session")
	}

	// Clean up.
	deleteResp, err := lib.AdminClient.DeleteDebugSessionWithResponse(
		t.Context(),
		"echo",
		sessionID,
		adminapi.RequestEditorFn(superRequestEditor),
	)
	lib.CheckErr(err, t)
	lib.VerifyStatusCode(deleteResp.StatusCode(), http.StatusNoContent, t)
}

// TestAdminDebugStopSessionNotFound verifies that stopping a non-existent session returns 404.
func TestAdminDebugStopSessionNotFound(t *testing.T) {
	t.Parallel()
	superRequestEditor := lib.SuperLogin(t)

	resp, err := lib.AdminClient.StopDebugSessionWithResponse(
		t.Context(),
		"echo",
		999999999,
		adminapi.RequestEditorFn(superRequestEditor),
	)
	lib.CheckErr(err, t)
	lib.VerifyStatusCode(resp.StatusCode(), http.StatusNotFound, t)
	lib.VerifyAdminAPIErrorResponse(resp.JSON404, t)
}

// --- DeleteDebugSession ---

// TestAdminDebugDeleteSession verifies that deleting a session returns 204 and the session
// is no longer retrievable.
func TestAdminDebugDeleteSession(t *testing.T) {
	superRequestEditor := lib.SuperLogin(t)

	sessionID := lib.StartDebugSession(t, superRequestEditor, "echo")

	deleteResp, err := lib.AdminClient.DeleteDebugSessionWithResponse(
		t.Context(),
		"echo",
		sessionID,
		adminapi.RequestEditorFn(superRequestEditor),
	)
	lib.CheckErr(err, t)
	lib.VerifyStatusCode(deleteResp.StatusCode(), http.StatusNoContent, t)

	// Verify the session is gone.
	getResp, err := lib.AdminClient.GetDebugSessionWithResponse(
		t.Context(),
		"echo",
		sessionID,
		adminapi.RequestEditorFn(superRequestEditor),
	)
	lib.CheckErr(err, t)
	lib.VerifyStatusCode(getResp.StatusCode(), http.StatusNotFound, t)
}

// TestAdminDebugDeleteSessionNotFound verifies that deleting a non-existent session returns 404.
func TestAdminDebugDeleteSessionNotFound(t *testing.T) {
	t.Parallel()
	superRequestEditor := lib.SuperLogin(t)

	resp, err := lib.AdminClient.DeleteDebugSessionWithResponse(
		t.Context(),
		"echo",
		999999999,
		adminapi.RequestEditorFn(superRequestEditor),
	)
	lib.CheckErr(err, t)
	lib.VerifyStatusCode(resp.StatusCode(), http.StatusNotFound, t)
	lib.VerifyAdminAPIErrorResponse(resp.JSON404, t)
}

// --- ListDebugSessionCalls & GetDebugSessionCall ---

// TestAdminDebugListSessionCallsWithTransitions verifies that after a gateway request is made
// during an active debug session, the call is recorded and flow transitions are populated
// when includeTransitions=true.
func TestAdminDebugListSessionCallsWithTransitions(t *testing.T) {
	superRequestEditor := lib.SuperLogin(t)

	sessionID := lib.StartDebugSession(t, superRequestEditor, "echo")
	defer func() {
		deleteResp, err := lib.AdminClient.DeleteDebugSessionWithResponse(
			t.Context(),
			"echo",
			sessionID,
			adminapi.RequestEditorFn(superRequestEditor),
		)
		lib.CheckErr(err, t)
		lib.VerifyStatusCode(deleteResp.StatusCode(), http.StatusNoContent, t)
	}()

	lib.MakeGatewayRequest(t, "echo", "/hi")

	listResp, err := lib.AdminClient.ListDebugSessionCallsWithResponse(
		t.Context(),
		"echo",
		sessionID,
		&adminapi.ListDebugSessionCallsParams{IncludeTransitions: true},
		adminapi.RequestEditorFn(superRequestEditor),
	)
	lib.CheckErr(err, t)
	lib.VerifyStatusCode(listResp.StatusCode(), http.StatusOK, t)

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

// TestAdminDebugListSessionCallsWithoutTransitions verifies that when includeTransitions=false,
// FlowTransitions are not included in the response.
func TestAdminDebugListSessionCallsWithoutTransitions(t *testing.T) {
	superRequestEditor := lib.SuperLogin(t)

	sessionID := lib.StartDebugSession(t, superRequestEditor, "echo")
	defer func() {
		deleteResp, err := lib.AdminClient.DeleteDebugSessionWithResponse(
			t.Context(),
			"echo",
			sessionID,
			adminapi.RequestEditorFn(superRequestEditor),
		)
		lib.CheckErr(err, t)
		lib.VerifyStatusCode(deleteResp.StatusCode(), http.StatusNoContent, t)
	}()

	lib.MakeGatewayRequest(t, "echo", "/hi")

	listResp, err := lib.AdminClient.ListDebugSessionCallsWithResponse(
		t.Context(),
		"echo",
		sessionID,
		&adminapi.ListDebugSessionCallsParams{IncludeTransitions: false},
		adminapi.RequestEditorFn(superRequestEditor),
	)
	lib.CheckErr(err, t)
	lib.VerifyStatusCode(listResp.StatusCode(), http.StatusOK, t)

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

// TestAdminDebugGetSessionCall verifies that a specific recorded call can be retrieved by ID.
func TestAdminDebugGetSessionCall(t *testing.T) {
	superRequestEditor := lib.SuperLogin(t)

	sessionID := lib.StartDebugSession(t, superRequestEditor, "echo")
	defer func() {
		deleteResp, err := lib.AdminClient.DeleteDebugSessionWithResponse(
			t.Context(),
			"echo",
			sessionID,
			adminapi.RequestEditorFn(superRequestEditor),
		)
		lib.CheckErr(err, t)
		lib.VerifyStatusCode(deleteResp.StatusCode(), http.StatusNoContent, t)
	}()

	lib.MakeGatewayRequest(t, "echo", "/hi")

	listResp, err := lib.AdminClient.ListDebugSessionCallsWithResponse(
		t.Context(),
		"echo",
		sessionID,
		&adminapi.ListDebugSessionCallsParams{IncludeTransitions: false},
		adminapi.RequestEditorFn(superRequestEditor),
	)
	lib.CheckErr(err, t)
	lib.VerifyStatusCode(listResp.StatusCode(), http.StatusOK, t)

	if listResp.JSON200 == nil || len(*listResp.JSON200) == 0 {
		t.Fatal("expected at least one recorded call")
	}
	callID := (*listResp.JSON200)[0].Id

	getCallResp, err := lib.AdminClient.GetDebugSessionCallWithResponse(
		t.Context(),
		"echo",
		sessionID,
		callID,
		adminapi.RequestEditorFn(superRequestEditor),
	)
	lib.CheckErr(err, t)
	lib.VerifyStatusCode(getCallResp.StatusCode(), http.StatusOK, t)

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

// TestAdminDebugGetSessionCallNotFound verifies that requesting a non-existent call returns 404.
func TestAdminDebugGetSessionCallNotFound(t *testing.T) {
	superRequestEditor := lib.SuperLogin(t)

	sessionID := lib.StartDebugSession(t, superRequestEditor, "echo")
	defer func() {
		deleteResp, err := lib.AdminClient.DeleteDebugSessionWithResponse(
			t.Context(),
			"echo",
			sessionID,
			adminapi.RequestEditorFn(superRequestEditor),
		)
		lib.CheckErr(err, t)
		lib.VerifyStatusCode(deleteResp.StatusCode(), http.StatusNoContent, t)
	}()

	resp, err := lib.AdminClient.GetDebugSessionCallWithResponse(
		t.Context(),
		"echo",
		sessionID,
		999999999,
		adminapi.RequestEditorFn(superRequestEditor),
	)
	lib.CheckErr(err, t)
	lib.VerifyStatusCode(resp.StatusCode(), http.StatusNotFound, t)
	lib.VerifyAdminAPIErrorResponse(resp.JSON404, t)
}

// --- Full lifecycle ---

// TestAdminDebugFullFlow exercises the complete debug session lifecycle end-to-end:
// start → get → hit gateway (records a call) → list calls → get call → stop → delete.
func TestAdminDebugFullFlow(t *testing.T) {
	superRequestEditor := lib.SuperLogin(t)

	// Start.
	startResp, err := lib.AdminClient.StartDebugSessionWithResponse(
		t.Context(),
		"echo",
		adminapi.StartDebugSessionJSONRequestBody{},
		adminapi.RequestEditorFn(superRequestEditor),
	)
	lib.CheckErr(err, t)
	lib.VerifyStatusCode(startResp.StatusCode(), http.StatusOK, t)
	sessionID := startResp.JSON200.Id

	// Get.
	getResp, err := lib.AdminClient.GetDebugSessionWithResponse(
		t.Context(),
		"echo",
		sessionID,
		adminapi.RequestEditorFn(superRequestEditor),
	)
	lib.CheckErr(err, t)
	lib.VerifyStatusCode(getResp.StatusCode(), http.StatusOK, t)
	lib.Matches(getResp.JSON200.Id, sessionID, t)

	// Make a gateway request so a call gets recorded.
	lib.MakeGatewayRequest(t, "echo", "/hi")

	// List calls with transitions.
	listCallsResp, err := lib.AdminClient.ListDebugSessionCallsWithResponse(
		t.Context(),
		"echo",
		sessionID,
		&adminapi.ListDebugSessionCallsParams{IncludeTransitions: false},
		adminapi.RequestEditorFn(superRequestEditor),
	)
	lib.CheckErr(err, t)
	lib.VerifyStatusCode(listCallsResp.StatusCode(), http.StatusOK, t)
	if len(*listCallsResp.JSON200) == 0 {
		t.Fatal("expected at least one recorded call after gateway request")
	}
	callID := (*listCallsResp.JSON200)[0].Id

	// Get specific call.
	getCallResp, err := lib.AdminClient.GetDebugSessionCallWithResponse(
		t.Context(),
		"echo",
		sessionID,
		callID,
		adminapi.RequestEditorFn(superRequestEditor),
	)
	lib.CheckErr(err, t)
	lib.VerifyStatusCode(getCallResp.StatusCode(), http.StatusOK, t)
	lib.Matches(getCallResp.JSON200.Id, callID, t)
	lib.Matches(false, len(getCallResp.JSON200.FlowTransitions) == 0, t)
	lib.Matches(http.MethodGet, getCallResp.JSON200.Method, t)
	lib.Matches("/gw/backend/echo/hi", getCallResp.JSON200.Url, t)

	// Stop.
	stopResp, err := lib.AdminClient.StopDebugSessionWithResponse(
		t.Context(),
		"echo",
		sessionID,
		adminapi.RequestEditorFn(superRequestEditor),
	)
	lib.CheckErr(err, t)
	lib.VerifyStatusCode(stopResp.StatusCode(), http.StatusNoContent, t)

	// Delete.
	deleteResp, err := lib.AdminClient.DeleteDebugSessionWithResponse(
		t.Context(),
		"echo",
		sessionID,
		adminapi.RequestEditorFn(superRequestEditor),
	)
	lib.CheckErr(err, t)
	lib.VerifyStatusCode(deleteResp.StatusCode(), http.StatusNoContent, t)
}
