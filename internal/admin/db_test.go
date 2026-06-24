package admin

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"testing"
	"time"

	adminapi "github.com/trebent/kerberos/internal/oapi/admin"
)

// --- helpers ---

func mustCreateAdminUser(t *testing.T, username string) int64 {
	t.Helper()
	id, err := dbCreateUser(context.Background(), testClient, username, "salt", "hashed")
	if err != nil {
		t.Fatalf("dbCreateUser(%q) error: %v", username, err)
	}
	return id
}

func mustCreateAdminGroup(t *testing.T, name string) int64 {
	t.Helper()
	id, err := dbCreateGroup(context.Background(), testClient, name)
	if err != nil {
		t.Fatalf("dbCreateGroup(%q) error: %v", name, err)
	}
	return id
}

// uniqueName returns a name that is unique per test to avoid constraint violations.
func uniqueName(t *testing.T, prefix string) string {
	t.Helper()
	return fmt.Sprintf("%s-%d", prefix, time.Now().UnixNano())
}

// --- Debug session tests ---

func TestDebugSessions(t *testing.T) {
	t.Run("Create and get/list debug session", func(t *testing.T) {
		ctx := context.Background()

		// Truncate for postgres tests that automatically trim nanoseconds.
		expiresAt := time.Now().Add(1 * time.Hour).Truncate(time.Microsecond)

		sessionID, err := dbCreateDebugSession(ctx, testClient, "backend", expiresAt)
		if err != nil {
			t.Fatalf("Failed to create debug session: %v", err)
		}
		if sessionID <= 0 {
			t.Fatalf("Invalid session ID returned: %d", sessionID)
		}

		// one more
		_, err = dbCreateDebugSession(ctx, testClient, "backend", expiresAt)
		if err != nil {
			t.Fatalf("Failed to create debug session: %v", err)
		}

		session, err := dbGetDebugSession(ctx, testClient, "backend", sessionID)
		if err != nil {
			t.Fatalf("Failed to get debug session: %v", err)
		}
		if int64(session.Id) != sessionID {
			t.Fatalf("Expected session ID %d, got %d", sessionID, session.Id)
		}
		if session.Backend != "backend" {
			t.Fatalf("Expected backend 'backend', got '%s'", session.Backend)
		}

		if !session.ExpiresAt.Equal(expiresAt) {
			t.Fatalf("Expected expires_at %v, got %v", expiresAt, session.ExpiresAt)
		}

		sessions, err := dbListDebugSessions(ctx, testClient, "backend")
		if err != nil {
			t.Fatalf("Failed to list debug sessions: %v", err)
		}
		found := false
		for _, s := range sessions {
			if int64(s.Id) == sessionID {
				found = true
				break
			}
		}
		if !found {
			t.Fatalf("Created debug session ID %d not found in list", sessionID)
		}

		if len(sessions) < 2 {
			t.Fatalf("Expected at least 2 debug sessions in list, got %d", len(sessions))
		}
	})

	t.Run("Get non-existent debug session", func(t *testing.T) {
		ctx := context.Background()
		_, err := dbGetDebugSession(ctx, testClient, "backend", 999999)
		if !errors.Is(err, errRowNotFound) {
			t.Fatalf("Expected errNoDebugSession, got %v", err)
		}

		_, err = dbGetDebugSession(ctx, testClient, "baccckkkkeeennnddddd", 999999)
		if !errors.Is(err, errRowNotFound) {
			t.Fatalf("Expected errNoDebugSession, got %v", err)
		}
	})

	t.Run("Update debug session", func(t *testing.T) {
		ctx := context.Background()

		initialExpiresAt := time.Now().Add(1 * time.Hour).Truncate(time.Microsecond)
		sessionID, err := dbCreateDebugSession(ctx, testClient, "backend", initialExpiresAt)
		if err != nil {
			t.Fatalf("Failed to create debug session: %v", err)
		}

		session, err := dbGetDebugSession(ctx, testClient, "backend", sessionID)
		if err != nil {
			t.Fatalf("Failed to get debug session: %v", err)
		}

		session.ExpiresAt = time.Now().UTC().Add(1 * time.Hour).Truncate(time.Microsecond)

		if err := dbUpdateDebugSession(ctx, testClient, *session); err != nil {
			t.Fatalf("Failed to update debug session: %v", err)
		}

		updatedSession, err := dbGetDebugSession(ctx, testClient, "backend", sessionID)
		if err != nil {
			t.Fatalf("Failed to get updated debug session: %v", err)
		}
		if !updatedSession.ExpiresAt.Equal(session.ExpiresAt) {
			t.Fatalf("Expected updated expires_at %v, got %v", session.ExpiresAt, updatedSession.ExpiresAt)
		}

		stoppedAt := time.Now().UTC().Truncate(time.Microsecond)
		updatedSession.StoppedAt = &stoppedAt

		if err := dbUpdateDebugSession(ctx, testClient, *updatedSession); err != nil {
			t.Fatalf("Failed to update debug session with stopped_at: %v", err)
		}

		finalSession, err := dbGetDebugSession(ctx, testClient, "backend", sessionID)
		if err != nil {
			t.Fatalf("Failed to get final debug session: %v", err)
		}
		if finalSession.StoppedAt == nil || !finalSession.StoppedAt.Equal(stoppedAt) {
			t.Fatalf("Expected stopped_at %v, got %v", stoppedAt, finalSession.StoppedAt)
		}
	})

	t.Run("Delete debug session", func(t *testing.T) {
		ctx := context.Background()
		sessionID, err := dbCreateDebugSession(ctx, testClient, "backend", time.Now().Add(1*time.Hour).Truncate(time.Microsecond))
		if err != nil {
			t.Fatalf("Failed to create debug session: %v", err)
		}

		if err := dbDeleteDebugSession(ctx, testClient, "backend", sessionID); err != nil {
			t.Fatalf("Failed to delete debug session: %v", err)
		}

		_, err = dbGetDebugSession(ctx, testClient, "backend", sessionID)
		if !errors.Is(err, errRowNotFound) {
			t.Fatalf("Expected errNoDebugSession after delete, got %v", err)
		}
	})
}

func TestDebugSessionCalls(t *testing.T) {
	staticSessionID, err := dbCreateDebugSession(
		context.Background(),
		testClient,
		"backend",
		time.Now().Add(1*time.Hour).Truncate(time.Microsecond),
	)
	if err != nil {
		t.Fatalf("Failed to create debug session: %v", err)
	}

	t.Run("Create and list debug session calls", func(t *testing.T) {
		ctx := context.Background()

		callStart := time.Now().UTC().Truncate(time.Microsecond)
		callStop := callStart.Add(1 * time.Second).UTC().Truncate(time.Microsecond)
		transitionStart := callStart.Add(100 * time.Millisecond).UTC().Truncate(time.Microsecond)
		transitionStop := transitionStart.Add(500 * time.Millisecond).UTC().Truncate(time.Microsecond)

		call1 := adminapi.DebugSessionCall{
			StartedAt:  callStart,
			StoppedAt:  callStop,
			Url:        "/test1",
			Method:     http.MethodGet,
			StatusCode: http.StatusOK,
			FlowTransitions: []adminapi.FlowTransition{
				{
					Component: "component1",
					Direction: "inbound",
					StartedAt: transitionStart,
					StoppedAt: transitionStop,
					Result: adminapi.FlowTransitionResult{
						Outcome: "success",
					},
				},
				{
					Component: "component1",
					Direction: "outbound",
					StartedAt: transitionStart,
					StoppedAt: transitionStop,
					Result: adminapi.FlowTransitionResult{
						Outcome: "success",
					},
				},
			},
		}
		if _, err := dbCreateDebugSessionCall(ctx, testClient, staticSessionID, call1); err != nil {
			t.Fatalf("Failed to create debug session call 1: %v", err)
		}

		call2 := adminapi.DebugSessionCall{
			StartedAt:  callStart,
			StoppedAt:  callStop,
			Url:        "/test2",
			Method:     http.MethodPost,
			StatusCode: http.StatusCreated,
			FlowTransitions: []adminapi.FlowTransition{
				{
					Component: "component1",
					Direction: "inbound",
					StartedAt: transitionStart,
					StoppedAt: transitionStop,
					Result: adminapi.FlowTransitionResult{
						Outcome: "success",
					},
				},
				{
					Component: "component1",
					Direction: "outbound",
					StartedAt: transitionStart,
					StoppedAt: transitionStop,
					Result: adminapi.FlowTransitionResult{
						Outcome: "success",
					},
				},
			},
		}
		if _, err := dbCreateDebugSessionCall(ctx, testClient, staticSessionID, call2); err != nil {
			t.Fatalf("Failed to create debug session call 2: %v", err)
		}

		calls, err := dbListDebugSessionCalls(ctx, testClient, staticSessionID, true)
		if err != nil {
			t.Fatalf("Failed to list debug session calls: %v", err)
		}

		if len(calls) < 2 {
			t.Fatalf("Expected at least 2 debug session calls, got %d", len(calls))
		}

		for _, call := range calls {
			if !call.StartedAt.Equal(callStart) {
				t.Fatalf("Expected started_at %v, got %v", callStart, call.StartedAt)
			}

			if !call.StoppedAt.Equal(callStop) {
				t.Fatalf("Expected stopped_at %v, got %v", callStop, call.StoppedAt)
			}

			if len(call.FlowTransitions) != 2 {
				t.Fatal("Expected 2 flow transitions for each call")
			}

			for _, transition := range call.FlowTransitions {
				if transition.Component != "component1" {
					t.Fatalf("Expected transition component 'component1', got '%s'", transition.Component)
				}

				if transition.Direction != "inbound" && transition.Direction != "outbound" {
					t.Fatalf("Expected transition direction 'inbound' or 'outbound', got '%s'", transition.Direction)
				}

				if !transition.StartedAt.Equal(transitionStart) {
					t.Fatalf("Expected transition started_at %v, got %v", transitionStart, transition.StartedAt)
				}

				if !transition.StoppedAt.Equal(transitionStop) {
					t.Fatalf("Expected transition stopped_at %v, got %v", transitionStop, transition.StoppedAt)
				}

				if transition.Result.Outcome != "success" {
					t.Fatalf("Expected transition outcome 'success', got '%s'", transition.Result.Outcome)
				}

				if transition.Result.Cause != nil {
					t.Fatalf("Expected transition cause '', got '%v'", transition.Result.Cause)
				}
			}
		}
	})

	t.Run("Get debug session call by ID", func(t *testing.T) {
		ctx := context.Background()
		callID, err := dbCreateDebugSessionCall(ctx, testClient, staticSessionID, adminapi.DebugSessionCall{
			Method:     http.MethodGet,
			Url:        "/test-get",
			StartedAt:  time.Now().UTC().Truncate(time.Microsecond),
			StoppedAt:  time.Now().UTC().Add(1 * time.Second).Truncate(time.Microsecond),
			StatusCode: http.StatusOK,
		})
		if err != nil {
			t.Fatalf("Failed to create debug session call: %v", err)
		}

		call, err := dbGetDebugSessionCall(ctx, testClient, callID)
		if err != nil {
			t.Fatalf("Failed to get debug session call by ID: %v", err)
		}

		if int64(call.Id) != callID {
			t.Fatalf("Expected call ID %d, got %d", callID, call.Id)
		}

		if call.Method != http.MethodGet {
			t.Fatalf("Expected method 'GET', got '%s'", call.Method)
		}

		if call.Url != "/test-get" {
			t.Fatalf("Expected URL '/test-get', got '%s'", call.Url)
		}
	})
}

// --- Superuser ---

func TestDBGetSuperuser(t *testing.T) {
	ctx := context.Background()

	t.Run("get existing", func(t *testing.T) {
		// The superuser is bootstrapped by TestMain.
		u, err := dbGetSuperuser(ctx, testClient)
		if err != nil {
			t.Fatalf("dbGetSuperuser error: %v", err)
		}
		if u == nil {
			t.Fatal("expected non-nil superuser")
		}
		if u.Username != testClientID {
			t.Fatalf("expected username %q, got %q", testClientID, u.Username)
		}
	})
}

// --- Sessions ---

func TestDBSessions(t *testing.T) {
	ctx := context.Background()
	username := uniqueName(t, "session-user")
	userID := mustCreateAdminUser(t, username)

	t.Run("create and get", func(t *testing.T) {
		sessionID := uniqueName(t, "session")
		if err := dbCreateSession(ctx, testClient, userID, sessionID); err != nil {
			t.Fatalf("dbCreateSession error: %v", err)
		}

		s, err := dbGetSession(ctx, testClient, sessionID)
		if err != nil {
			t.Fatalf("dbGetSession error: %v", err)
		}
		if s.UserID != userID {
			t.Fatalf("expected UserID %d, got %d", userID, s.UserID)
		}
		if s.SessionID != sessionID {
			t.Fatalf("expected SessionID %q, got %q", sessionID, s.SessionID)
		}
	})

	t.Run("get not found", func(t *testing.T) {
		_, err := dbGetSession(ctx, testClient, "no-such-session")
		if !errors.Is(err, errRowNotFound) {
			t.Fatalf("expected errNoSession, got %v", err)
		}
	})

	t.Run("delete", func(t *testing.T) {
		sessionID := uniqueName(t, "session-del")
		if err := dbCreateSession(ctx, testClient, userID, sessionID); err != nil {
			t.Fatalf("dbCreateSession error: %v", err)
		}

		if err := dbDeleteSession(ctx, testClient, sessionID); err != nil {
			t.Fatalf("dbDeleteSession error: %v", err)
		}

		_, err := dbGetSession(ctx, testClient, sessionID)
		if !errors.Is(err, errRowNotFound) {
			t.Fatalf("expected errNoSession after delete, got %v", err)
		}
	})
}

// --- Users ---

func TestDBUsers(t *testing.T) {
	ctx := context.Background()

	t.Run("create", func(t *testing.T) {
		id, err := dbCreateUser(ctx, testClient, uniqueName(t, "user-create"), "salt", "hashed")
		if err != nil {
			t.Fatalf("dbCreateUser error: %v", err)
		}
		if id == 0 {
			t.Fatal("expected non-zero user ID")
		}
	})

	t.Run("create duplicate", func(t *testing.T) {
		name := uniqueName(t, "user-dup")
		mustCreateAdminUser(t, name)
		_, err := dbCreateUser(ctx, testClient, name, "salt", "hashed")
		if err == nil {
			t.Fatal("expected error for duplicate user name, got nil")
		}
	})

	t.Run("get", func(t *testing.T) {
		name := uniqueName(t, "user-get")
		userID := mustCreateAdminUser(t, name)

		u, err := dbGetUser(ctx, testClient, userID)
		if err != nil {
			t.Fatalf("dbGetUser error: %v", err)
		}
		if int64(u.Id) != userID {
			t.Fatalf("expected user ID %d, got %d", userID, u.Id)
		}
		if u.Username != name {
			t.Fatalf("expected username %q, got %q", name, u.Username)
		}
	})

	t.Run("get not found", func(t *testing.T) {
		_, err := dbGetUser(ctx, testClient, 999999)
		if !errors.Is(err, errRowNotFound) {
			t.Fatalf("expected errNoUser, got %v", err)
		}
	})

	t.Run("list", func(t *testing.T) {
		name := uniqueName(t, "user-list")
		mustCreateAdminUser(t, name)

		users, err := dbListUsers(ctx, testClient)
		if err != nil {
			t.Fatalf("dbListUsers error: %v", err)
		}
		found := false
		for _, u := range users {
			if u.Username == name {
				found = true
				break
			}
		}
		if !found {
			t.Fatalf("expected to find user %q in list", name)
		}
	})

	t.Run("update", func(t *testing.T) {
		name := uniqueName(t, "user-upd")
		userID := mustCreateAdminUser(t, name)

		newName := uniqueName(t, "user-upd-new")
		if err := dbUpdateUser(ctx, testClient, userID, newName); err != nil {
			t.Fatalf("dbUpdateUser error: %v", err)
		}

		u, err := dbGetUser(ctx, testClient, userID)
		if err != nil {
			t.Fatalf("dbGetUser after update error: %v", err)
		}
		if u.Username != newName {
			t.Fatalf("expected username %q after update, got %q", newName, u.Username)
		}
	})

	t.Run("delete", func(t *testing.T) {
		name := uniqueName(t, "user-del")
		userID := mustCreateAdminUser(t, name)

		if err := dbDeleteUser(ctx, testClient, userID); err != nil {
			t.Fatalf("dbDeleteUser error: %v", err)
		}

		_, err := dbGetUser(ctx, testClient, userID)
		if !errors.Is(err, errRowNotFound) {
			t.Fatalf("expected errNoUser after delete, got %v", err)
		}
	})

	t.Run("get user auth", func(t *testing.T) {
		name := uniqueName(t, "user-auth")
		userID, err := dbCreateUser(ctx, testClient, name, "mysalt", "myhashed")
		if err != nil {
			t.Fatalf("dbCreateUser error: %v", err)
		}

		auth, err := dbGetUserAuth(ctx, testClient, userID)
		if err != nil {
			t.Fatalf("dbGetUserAuth error: %v", err)
		}
		if auth.Salt != "mysalt" {
			t.Fatalf("expected salt %q, got %q", "mysalt", auth.Salt)
		}
		if auth.HashedPassword != "myhashed" {
			t.Fatalf("expected hashed password %q, got %q", "myhashed", auth.HashedPassword)
		}
	})

	t.Run("get user auth not found", func(t *testing.T) {
		_, err := dbGetUserAuth(ctx, testClient, 999999)
		if !errors.Is(err, errRowNotFound) {
			t.Fatalf("expected errNoUser, got %v", err)
		}
	})

	t.Run("update password", func(t *testing.T) {
		name := uniqueName(t, "user-pw")
		userID, err := dbCreateUser(ctx, testClient, name, "salt", "oldpassword")
		if err != nil {
			t.Fatalf("dbCreateUser error: %v", err)
		}

		if err := dbUpdateUserPassword(ctx, testClient, userID, "newsalt", "newpassword"); err != nil {
			t.Fatalf("dbUpdateUserPassword error: %v", err)
		}

		auth, err := dbGetUserAuth(ctx, testClient, userID)
		if err != nil {
			t.Fatalf("dbGetUserAuth after password update error: %v", err)
		}
		if auth.Salt != "newsalt" {
			t.Fatalf("expected salt %q after update, got %q", "newsalt", auth.Salt)
		}
		if auth.HashedPassword != "newpassword" {
			t.Fatalf("expected hashed password %q after update, got %q", "newpassword", auth.HashedPassword)
		}
	})

	t.Run("login lookup", func(t *testing.T) {
		name := uniqueName(t, "user-login")
		userID, err := dbCreateUser(ctx, testClient, name, "salt", "hashed")
		if err != nil {
			t.Fatalf("dbCreateUser error: %v", err)
		}

		lu, err := dbLoginLookup(ctx, testClient, name)
		if err != nil {
			t.Fatalf("dbLoginLookup error: %v", err)
		}
		if lu.ID != userID {
			t.Fatalf("expected user ID %d, got %d", userID, lu.ID)
		}
		if lu.Salt != "salt" {
			t.Fatalf("expected salt %q, got %q", "salt", lu.Salt)
		}
		if lu.HashedPassword != "hashed" {
			t.Fatalf("expected hashed password %q, got %q", "hashed", lu.HashedPassword)
		}
	})

	t.Run("login lookup not found", func(t *testing.T) {
		_, err := dbLoginLookup(ctx, testClient, "no-such-user")
		if !errors.Is(err, errRowNotFound) {
			t.Fatalf("expected errNoUser, got %v", err)
		}
	})
}

// --- Groups ---

func TestDBGroups(t *testing.T) {
	ctx := context.Background()

	t.Run("create", func(t *testing.T) {
		id, err := dbCreateGroup(ctx, testClient, uniqueName(t, "group-create"))
		if err != nil {
			t.Fatalf("dbCreateGroup error: %v", err)
		}
		if id == 0 {
			t.Fatal("expected non-zero group ID")
		}
	})

	t.Run("create duplicate", func(t *testing.T) {
		name := uniqueName(t, "group-dup")
		mustCreateAdminGroup(t, name)
		_, err := dbCreateGroup(ctx, testClient, name)
		if err == nil {
			t.Fatal("expected error for duplicate group name, got nil")
		}
	})

	t.Run("get", func(t *testing.T) {
		name := uniqueName(t, "group-get")
		groupID := mustCreateAdminGroup(t, name)

		g, err := dbGetGroup(ctx, testClient, groupID)
		if err != nil {
			t.Fatalf("dbGetGroup error: %v", err)
		}
		if int64(g.Id) != groupID {
			t.Fatalf("expected group ID %d, got %d", groupID, g.Id)
		}
		if g.Name != name {
			t.Fatalf("expected group name %q, got %q", name, g.Name)
		}
	})

	t.Run("get not found", func(t *testing.T) {
		_, err := dbGetGroup(ctx, testClient, 999999)
		if !errors.Is(err, errRowNotFound) {
			t.Fatalf("expected errNoGroup, got %v", err)
		}
	})

	t.Run("list", func(t *testing.T) {
		name := uniqueName(t, "group-list")
		mustCreateAdminGroup(t, name)

		groups, err := dbListGroups(ctx, testClient)
		if err != nil {
			t.Fatalf("dbListGroups error: %v", err)
		}
		found := false
		for _, g := range groups {
			if g.Name == name {
				found = true
				break
			}
		}
		if !found {
			t.Fatalf("expected to find group %q in list", name)
		}
	})

	t.Run("update", func(t *testing.T) {
		name := uniqueName(t, "group-upd")
		groupID := mustCreateAdminGroup(t, name)

		newName := uniqueName(t, "group-upd-new")
		if err := dbUpdateGroup(ctx, testClient, groupID, newName); err != nil {
			t.Fatalf("dbUpdateGroup error: %v", err)
		}

		g, err := dbGetGroup(ctx, testClient, groupID)
		if err != nil {
			t.Fatalf("dbGetGroup after update error: %v", err)
		}
		if g.Name != newName {
			t.Fatalf("expected group name %q after update, got %q", newName, g.Name)
		}
	})

	t.Run("delete", func(t *testing.T) {
		name := uniqueName(t, "group-del")
		groupID := mustCreateAdminGroup(t, name)

		if err := dbDeleteGroup(ctx, testClient, groupID); err != nil {
			t.Fatalf("dbDeleteGroup error: %v", err)
		}

		_, err := dbGetGroup(ctx, testClient, groupID)
		if !errors.Is(err, errRowNotFound) {
			t.Fatalf("expected errNoGroup after delete, got %v", err)
		}
	})
}

// --- Cascade deletes ---

// TestDBCascadeDeletes verifies that ON DELETE CASCADE behaviour defined in the SQL schema
// is enforced for all foreign-key relationships in the admin_* tables.
func TestDBCascadeDeletes(t *testing.T) {
	ctx := context.Background()

	t.Run("delete user cascades sessions", func(t *testing.T) {
		userID := mustCreateAdminUser(t, uniqueName(t, "cascade-sess-user"))
		sessionID := uniqueName(t, "cascade-sess")
		if err := dbCreateSession(ctx, testClient, userID, sessionID); err != nil {
			t.Fatalf("dbCreateSession error: %v", err)
		}

		// Verify session exists.
		if _, err := dbGetSession(ctx, testClient, sessionID); err != nil {
			t.Fatalf("dbGetSession before delete error: %v", err)
		}

		// Delete user — should cascade to sessions.
		if err := dbDeleteUser(ctx, testClient, userID); err != nil {
			t.Fatalf("dbDeleteUser error: %v", err)
		}

		_, err := dbGetSession(ctx, testClient, sessionID)
		if !errors.Is(err, errRowNotFound) {
			t.Fatalf("expected errNoSession after user cascade delete, got %v", err)
		}
	})

	t.Run("delete user cascades group bindings", func(t *testing.T) {
		userID := mustCreateAdminUser(t, uniqueName(t, "cascade-bind-user"))
		groupID := mustCreateAdminGroup(t, uniqueName(t, "cascade-bind-grp-u"))
		if err := dbUpdateUserGroupBindings(ctx, testClient, userID, []int{int(groupID)}); err != nil {
			t.Fatalf("dbUpdateUserGroupBindings error: %v", err)
		}

		bindings, err := dbListGroupBindings(ctx, testClient, userID)
		if err != nil {
			t.Fatalf("dbListGroupBindings error: %v", err)
		}
		if len(bindings) != 1 {
			t.Fatalf("expected 1 binding, got %d", len(bindings))
		}

		// Delete user — should cascade to group bindings.
		if err := dbDeleteUser(ctx, testClient, userID); err != nil {
			t.Fatalf("dbDeleteUser error: %v", err)
		}

		bindings, err = dbListGroupBindings(ctx, testClient, userID)
		if err != nil {
			t.Fatalf("dbListGroupBindings after user delete error: %v", err)
		}
		if len(bindings) != 0 {
			t.Fatalf("expected 0 bindings after user cascade delete, got %d", len(bindings))
		}
	})

	t.Run("delete group cascades group bindings", func(t *testing.T) {
		userID := mustCreateAdminUser(t, uniqueName(t, "cascade-grp-user"))
		groupID := mustCreateAdminGroup(t, uniqueName(t, "cascade-grp"))
		if err := dbUpdateUserGroupBindings(ctx, testClient, userID, []int{int(groupID)}); err != nil {
			t.Fatalf("dbUpdateUserGroupBindings error: %v", err)
		}

		bindings, err := dbListGroupBindings(ctx, testClient, userID)
		if err != nil {
			t.Fatalf("dbListGroupBindings error: %v", err)
		}
		if len(bindings) != 1 {
			t.Fatalf("expected 1 binding, got %d", len(bindings))
		}

		// Delete group — should cascade to group bindings.
		if err := dbDeleteGroup(ctx, testClient, groupID); err != nil {
			t.Fatalf("dbDeleteGroup error: %v", err)
		}

		bindings, err = dbListGroupBindings(ctx, testClient, userID)
		if err != nil {
			t.Fatalf("dbListGroupBindings after group delete error: %v", err)
		}
		if len(bindings) != 0 {
			t.Fatalf("expected 0 bindings after group cascade delete, got %d", len(bindings))
		}
	})
}

// --- Group bindings ---

func TestDBGroupBindings(t *testing.T) {
	ctx := context.Background()

	t.Run("list and update bindings", func(t *testing.T) {
		userID := mustCreateAdminUser(t, uniqueName(t, "binding-user"))
		groupID1 := mustCreateAdminGroup(t, uniqueName(t, "binding-grp1"))
		groupID2 := mustCreateAdminGroup(t, uniqueName(t, "binding-grp2"))

		// Initially no bindings.
		bindings, err := dbListGroupBindings(ctx, testClient, userID)
		if err != nil {
			t.Fatalf("dbListGroupBindings error: %v", err)
		}
		if len(bindings) != 0 {
			t.Fatalf("expected 0 bindings, got %d", len(bindings))
		}

		// Set two groups.
		if err := dbUpdateUserGroupBindings(ctx, testClient, userID, []int{int(groupID1), int(groupID2)}); err != nil {
			t.Fatalf("dbUpdateUserGroupBindings error: %v", err)
		}

		bindings, err = dbListGroupBindings(ctx, testClient, userID)
		if err != nil {
			t.Fatalf("dbListGroupBindings after update error: %v", err)
		}
		if len(bindings) != 2 {
			t.Fatalf("expected 2 bindings, got %d", len(bindings))
		}

		// Reduce to one group.
		if err := dbUpdateUserGroupBindings(ctx, testClient, userID, []int{int(groupID1)}); err != nil {
			t.Fatalf("dbUpdateUserGroupBindings (reduce) error: %v", err)
		}

		bindings, err = dbListGroupBindings(ctx, testClient, userID)
		if err != nil {
			t.Fatalf("dbListGroupBindings after reduce error: %v", err)
		}
		if len(bindings) != 1 {
			t.Fatalf("expected 1 binding, got %d", len(bindings))
		}
		if bindings[0].GroupID != groupID1 {
			t.Fatalf("expected groupID %d, got %d", groupID1, bindings[0].GroupID)
		}

		// Clear all groups.
		if err := dbUpdateUserGroupBindings(ctx, testClient, userID, []int{}); err != nil {
			t.Fatalf("dbUpdateUserGroupBindings (clear) error: %v", err)
		}

		bindings, err = dbListGroupBindings(ctx, testClient, userID)
		if err != nil {
			t.Fatalf("dbListGroupBindings after clear error: %v", err)
		}
		if len(bindings) != 0 {
			t.Fatalf("expected 0 bindings after clear, got %d", len(bindings))
		}
	})
}
