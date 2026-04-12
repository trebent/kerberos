package basic

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"
)

// --- helpers ---

func mustCreateOrg(t *testing.T, name string) (orgID, adminUserID int64) {
	t.Helper()
	orgID, adminUserID, _, _, err := dbCreateOrganisation(context.Background(), testClient, name)
	if err != nil {
		t.Fatalf("dbCreateOrganisation(%q) error: %v", name, err)
	}
	return orgID, adminUserID
}

func mustCreateGroup(t *testing.T, orgID int64, name string) int64 {
	t.Helper()
	id, err := dbCreateGroup(context.Background(), testClient, orgID, name)
	if err != nil {
		t.Fatalf("dbCreateGroup(org=%d, %q) error: %v", orgID, name, err)
	}
	return id
}

func mustCreateUser(t *testing.T, orgID int64, name string) int64 {
	t.Helper()
	id, err := dbCreateUser(context.Background(), testClient, name, "salt", "hashed", orgID)
	if err != nil {
		t.Fatalf("dbCreateUser(org=%d, %q) error: %v", orgID, name, err)
	}
	return id
}

// uniqueName returns a name that is unique per test to avoid constraint violations.
func uniqueName(t *testing.T, prefix string) string {
	t.Helper()
	return fmt.Sprintf("%s-%d", prefix, time.Now().UnixNano())
}

// --- Organisations ---

func TestDBOrganisations(t *testing.T) {
	ctx := context.Background()

	t.Run("create", func(t *testing.T) {
		name := uniqueName(t, "org-create")
		orgID, adminUserID, adminUsername, adminPassword, err := dbCreateOrganisation(ctx, testClient, name)
		if err != nil {
			t.Fatalf("dbCreateOrganisation error: %v", err)
		}
		if orgID == 0 {
			t.Fatal("expected non-zero orgID")
		}
		if adminUserID == 0 {
			t.Fatal("expected non-zero adminUserID")
		}
		if adminUsername == "" {
			t.Fatal("expected non-empty adminUsername")
		}
		if adminPassword == "" {
			t.Fatal("expected non-empty adminPassword")
		}
	})

	t.Run("create duplicate", func(t *testing.T) {
		name := uniqueName(t, "org-dup")
		mustCreateOrg(t, name)
		_, _, _, _, err := dbCreateOrganisation(ctx, testClient, name)
		if err == nil {
			t.Fatal("expected error for duplicate org name, got nil")
		}
	})

	t.Run("get", func(t *testing.T) {
		name := uniqueName(t, "org-get")
		orgID, _ := mustCreateOrg(t, name)

		org, err := dbGetOrg(ctx, testClient, orgID)
		if err != nil {
			t.Fatalf("dbGetOrg error: %v", err)
		}
		if org.Id != orgID {
			t.Fatalf("expected org ID %d, got %d", orgID, org.Id)
		}
		if org.Name != name {
			t.Fatalf("expected org name %q, got %q", name, org.Name)
		}
	})

	t.Run("get not found", func(t *testing.T) {
		_, err := dbGetOrg(ctx, testClient, 999999)
		if !errors.Is(err, errNoOrg) {
			t.Fatalf("expected errNoOrg, got %v", err)
		}
	})

	t.Run("list", func(t *testing.T) {
		name := uniqueName(t, "org-list")
		mustCreateOrg(t, name)

		orgs, err := dbListOrgs(ctx, testClient)
		if err != nil {
			t.Fatalf("dbListOrgs error: %v", err)
		}
		found := false
		for _, o := range orgs {
			if o.Name == name {
				found = true
				break
			}
		}
		if !found {
			t.Fatalf("expected to find org %q in list", name)
		}
	})

	t.Run("update", func(t *testing.T) {
		name := uniqueName(t, "org-upd")
		orgID, _ := mustCreateOrg(t, name)

		newName := uniqueName(t, "org-upd-new")
		if err := dbUpdateOrg(ctx, testClient, orgID, newName); err != nil {
			t.Fatalf("dbUpdateOrg error: %v", err)
		}

		org, err := dbGetOrg(ctx, testClient, orgID)
		if err != nil {
			t.Fatalf("dbGetOrg after update error: %v", err)
		}
		if org.Name != newName {
			t.Fatalf("expected name %q after update, got %q", newName, org.Name)
		}
	})

	t.Run("delete", func(t *testing.T) {
		name := uniqueName(t, "org-del")
		orgID, _ := mustCreateOrg(t, name)

		if err := dbDeleteOrg(ctx, testClient, orgID); err != nil {
			t.Fatalf("dbDeleteOrg error: %v", err)
		}

		_, err := dbGetOrg(ctx, testClient, orgID)
		if !errors.Is(err, errNoOrg) {
			t.Fatalf("expected errNoOrg after delete, got %v", err)
		}
	})
}

// --- Users ---

func TestDBUsers(t *testing.T) {
	ctx := context.Background()
	orgName := uniqueName(t, "org-users")
	orgID, _ := mustCreateOrg(t, orgName)

	t.Run("create", func(t *testing.T) {
		id, err := dbCreateUser(ctx, testClient, uniqueName(t, "user-create"), "salt", "hashed", orgID)
		if err != nil {
			t.Fatalf("dbCreateUser error: %v", err)
		}
		if id == 0 {
			t.Fatal("expected non-zero user ID")
		}
	})

	t.Run("create duplicate", func(t *testing.T) {
		name := uniqueName(t, "user-dup")
		mustCreateUser(t, orgID, name)
		_, err := dbCreateUser(ctx, testClient, name, "salt", "hashed", orgID)
		if err == nil {
			t.Fatal("expected error for duplicate user name, got nil")
		}
	})

	t.Run("get", func(t *testing.T) {
		name := uniqueName(t, "user-get")
		userID := mustCreateUser(t, orgID, name)

		u, err := dbGetUser(ctx, testClient, orgID, userID)
		if err != nil {
			t.Fatalf("dbGetUser error: %v", err)
		}
		if u.Id != userID {
			t.Fatalf("expected user ID %d, got %d", userID, u.Id)
		}
		if u.Name != name {
			t.Fatalf("expected user name %q, got %q", name, u.Name)
		}
	})

	t.Run("get not found", func(t *testing.T) {
		_, err := dbGetUser(ctx, testClient, orgID, 999999)
		if !errors.Is(err, errNoUser) {
			t.Fatalf("expected errNoUser, got %v", err)
		}
	})

	t.Run("list", func(t *testing.T) {
		name := uniqueName(t, "user-list")
		mustCreateUser(t, orgID, name)

		users, err := dbListUsers(ctx, testClient, orgID)
		if err != nil {
			t.Fatalf("dbListUsers error: %v", err)
		}
		found := false
		for _, u := range users {
			if u.Name == name {
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
		userID := mustCreateUser(t, orgID, name)

		newName := uniqueName(t, "user-upd-new")
		if err := dbUpdateUser(ctx, testClient, orgID, userID, newName); err != nil {
			t.Fatalf("dbUpdateUser error: %v", err)
		}

		u, err := dbGetUser(ctx, testClient, orgID, userID)
		if err != nil {
			t.Fatalf("dbGetUser after update error: %v", err)
		}
		if u.Name != newName {
			t.Fatalf("expected name %q after update, got %q", newName, u.Name)
		}
	})

	t.Run("delete", func(t *testing.T) {
		name := uniqueName(t, "user-del")
		userID := mustCreateUser(t, orgID, name)

		if err := dbDeleteUser(ctx, testClient, orgID, userID); err != nil {
			t.Fatalf("dbDeleteUser error: %v", err)
		}

		_, err := dbGetUser(ctx, testClient, orgID, userID)
		if !errors.Is(err, errNoUser) {
			t.Fatalf("expected errNoUser after delete, got %v", err)
		}
	})

	t.Run("get user auth", func(t *testing.T) {
		name := uniqueName(t, "user-auth")
		userID, err := dbCreateUser(ctx, testClient, name, "salt", "hashed", orgID)
		if err != nil {
			t.Fatalf("dbCreateUser error: %v", err)
		}

		auth, err := dbGetUserAuth(ctx, testClient, orgID, userID)
		if err != nil {
			t.Fatalf("dbGetUserAuth error: %v", err)
		}
		if auth.Salt != "salt" {
			t.Fatalf("expected salt %q, got %q", "salt", auth.Salt)
		}
		if auth.HashedPassword != "hashed" {
			t.Fatalf("expected hashed password %q, got %q", "hashed", auth.HashedPassword)
		}
	})

	t.Run("get user auth not found", func(t *testing.T) {
		_, err := dbGetUserAuth(ctx, testClient, orgID, 999999)
		if !errors.Is(err, errNoUser) {
			t.Fatalf("expected errNoUser, got %v", err)
		}
	})

	t.Run("update password", func(t *testing.T) {
		name := uniqueName(t, "user-pw")
		oldPw := "oldpassword"
		userID, err := dbCreateUser(ctx, testClient, name, "salt", oldPw, orgID)
		if err != nil {
			t.Fatalf("dbCreateUser error: %v", err)
		}

		newPw := "newpassword"
		if err := dbUpdateUserPassword(ctx, testClient, userID, "salt", newPw); err != nil {
			t.Fatalf("dbUpdateUserPassword error: %v", err)
		}

		auth, err := dbGetUserAuth(ctx, testClient, orgID, userID)
		if err != nil {
			t.Fatalf("dbGetUserAuth after password update error: %v", err)
		}
		if auth.Salt != "salt" {
			t.Fatalf("expected salt %q after password update, got %q", "salt", auth.Salt)
		}
		if auth.HashedPassword != "newpassword" {
			t.Fatal("updated password does not match")
		}
		if auth.HashedPassword == oldPw {
			t.Fatal("old password should no longer match after update")
		}
	})

	t.Run("login lookup", func(t *testing.T) {
		name := uniqueName(t, "user-login")
		userID, err := dbCreateUser(ctx, testClient, name, "salt", "hashed", orgID)
		if err != nil {
			t.Fatalf("dbCreateUser error: %v", err)
		}

		lu, err := dbLoginLookup(ctx, testClient, orgID, name)
		if err != nil {
			t.Fatalf("dbLoginLookup error: %v", err)
		}
		if lu.ID != userID {
			t.Fatalf("expected user ID %d, got %d", userID, lu.ID)
		}
		if lu.OrganisationID != orgID {
			t.Fatalf("expected org ID %d, got %d", orgID, lu.OrganisationID)
		}
		if lu.Salt != "salt" {
			t.Fatalf("expected salt %q, got %q", "salt", lu.Salt)
		}
		if lu.HashedPassword != "hashed" {
			t.Fatalf("expected hashed password %q, got %q", "hashed", lu.HashedPassword)
		}
	})

	t.Run("login lookup not found", func(t *testing.T) {
		_, err := dbLoginLookup(ctx, testClient, orgID, "no-such-user")
		if !errors.Is(err, errNoUser) {
			t.Fatalf("expected errNoUser, got %v", err)
		}
	})
}

// --- Groups ---

func TestDBGroups(t *testing.T) {
	ctx := context.Background()
	orgName := uniqueName(t, "org-groups")
	orgID, _ := mustCreateOrg(t, orgName)

	t.Run("create", func(t *testing.T) {
		id, err := dbCreateGroup(ctx, testClient, orgID, uniqueName(t, "group-create"))
		if err != nil {
			t.Fatalf("dbCreateGroup error: %v", err)
		}
		if id == 0 {
			t.Fatal("expected non-zero group ID")
		}
	})

	t.Run("create duplicate", func(t *testing.T) {
		name := uniqueName(t, "group-dup")
		mustCreateGroup(t, orgID, name)
		_, err := dbCreateGroup(ctx, testClient, orgID, name)
		if err == nil {
			t.Fatal("expected error for duplicate group name, got nil")
		}
	})

	t.Run("get", func(t *testing.T) {
		name := uniqueName(t, "group-get")
		groupID := mustCreateGroup(t, orgID, name)

		g, err := dbGetGroup(ctx, testClient, orgID, groupID)
		if err != nil {
			t.Fatalf("dbGetGroup error: %v", err)
		}
		if g.Id != groupID {
			t.Fatalf("expected group ID %d, got %d", groupID, g.Id)
		}
		if g.Name != name {
			t.Fatalf("expected group name %q, got %q", name, g.Name)
		}
	})

	t.Run("get not found", func(t *testing.T) {
		_, err := dbGetGroup(ctx, testClient, orgID, 999999)
		if !errors.Is(err, errNoGroup) {
			t.Fatalf("expected errNoGroup, got %v", err)
		}
	})

	t.Run("list", func(t *testing.T) {
		name := uniqueName(t, "group-list")
		mustCreateGroup(t, orgID, name)

		groups, err := dbListGroups(ctx, testClient, orgID)
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
		groupID := mustCreateGroup(t, orgID, name)

		newName := uniqueName(t, "group-upd-new")
		if err := dbUpdateGroup(ctx, testClient, orgID, groupID, newName); err != nil {
			t.Fatalf("dbUpdateGroup error: %v", err)
		}

		g, err := dbGetGroup(ctx, testClient, orgID, groupID)
		if err != nil {
			t.Fatalf("dbGetGroup after update error: %v", err)
		}
		if g.Name != newName {
			t.Fatalf("expected name %q after update, got %q", newName, g.Name)
		}
	})

	t.Run("delete", func(t *testing.T) {
		name := uniqueName(t, "group-del")
		groupID := mustCreateGroup(t, orgID, name)

		if err := dbDeleteGroup(ctx, testClient, orgID, groupID); err != nil {
			t.Fatalf("dbDeleteGroup error: %v", err)
		}

		_, err := dbGetGroup(ctx, testClient, orgID, groupID)
		if !errors.Is(err, errNoGroup) {
			t.Fatalf("expected errNoGroup after delete, got %v", err)
		}
	})
}

// --- Group Bindings ---

func TestDBGroupBindings(t *testing.T) {
	ctx := context.Background()
	orgName := uniqueName(t, "org-bindings")
	orgID, _ := mustCreateOrg(t, orgName)
	userID := mustCreateUser(t, orgID, uniqueName(t, "user-bindings"))
	group1Name := uniqueName(t, "group-bind-1")
	group2Name := uniqueName(t, "group-bind-2")
	mustCreateGroup(t, orgID, group1Name)
	mustCreateGroup(t, orgID, group2Name)

	t.Run("list empty", func(t *testing.T) {
		bindings, err := dbListGroupBindings(ctx, testClient, orgID, userID)
		if err != nil {
			t.Fatalf("dbListGroupBindings error: %v", err)
		}
		if len(bindings) != 0 {
			t.Fatalf("expected 0 bindings, got %d", len(bindings))
		}
	})

	t.Run("update adds bindings", func(t *testing.T) {
		if err := dbUpdateUserGroupBindings(ctx, testClient, orgID, userID, []string{group1Name, group2Name}); err != nil {
			t.Fatalf("dbUpdateUserGroupBindings error: %v", err)
		}

		bindings, err := dbListGroupBindings(ctx, testClient, orgID, userID)
		if err != nil {
			t.Fatalf("dbListGroupBindings error: %v", err)
		}
		if len(bindings) != 2 {
			t.Fatalf("expected 2 bindings, got %d", len(bindings))
		}
		names := make(map[string]bool)
		for _, b := range bindings {
			names[b.Name] = true
		}
		if !names[group1Name] || !names[group2Name] {
			t.Fatalf("expected bindings for %q and %q", group1Name, group2Name)
		}
	})

	t.Run("update removes bindings", func(t *testing.T) {
		// Remove group2, keep only group1.
		if err := dbUpdateUserGroupBindings(ctx, testClient, orgID, userID, []string{group1Name}); err != nil {
			t.Fatalf("dbUpdateUserGroupBindings error: %v", err)
		}

		bindings, err := dbListGroupBindings(ctx, testClient, orgID, userID)
		if err != nil {
			t.Fatalf("dbListGroupBindings error: %v", err)
		}
		if len(bindings) != 1 {
			t.Fatalf("expected 1 binding, got %d", len(bindings))
		}
		if bindings[0].Name != group1Name {
			t.Fatalf("expected binding for %q, got %q", group1Name, bindings[0].Name)
		}
	})

	t.Run("update clears all bindings", func(t *testing.T) {
		if err := dbUpdateUserGroupBindings(ctx, testClient, orgID, userID, []string{}); err != nil {
			t.Fatalf("dbUpdateUserGroupBindings error: %v", err)
		}

		bindings, err := dbListGroupBindings(ctx, testClient, orgID, userID)
		if err != nil {
			t.Fatalf("dbListGroupBindings error: %v", err)
		}
		if len(bindings) != 0 {
			t.Fatalf("expected 0 bindings after clearing, got %d", len(bindings))
		}
	})

	t.Run("get user group names", func(t *testing.T) {
		if err := dbUpdateUserGroupBindings(ctx, testClient, orgID, userID, []string{group1Name, group2Name}); err != nil {
			t.Fatalf("dbUpdateUserGroupBindings error: %v", err)
		}

		names, err := dbGetUserGroupNames(ctx, testClient, orgID, userID)
		if err != nil {
			t.Fatalf("dbGetUserGroupNames error: %v", err)
		}
		if len(names) != 2 {
			t.Fatalf("expected 2 group names, got %d", len(names))
		}
		nameSet := make(map[string]bool)
		for _, n := range names {
			nameSet[n] = true
		}
		if !nameSet[group1Name] || !nameSet[group2Name] {
			t.Fatalf("expected group names %q and %q", group1Name, group2Name)
		}
	})
}

// --- Sessions ---

func TestDBSessions(t *testing.T) {
	ctx := context.Background()
	orgName := uniqueName(t, "org-sessions")
	orgID, _ := mustCreateOrg(t, orgName)
	userID := mustCreateUser(t, orgID, uniqueName(t, "user-sessions"))

	t.Run("create and get", func(t *testing.T) {
		sessionID := fmt.Sprintf("test-session-%d", time.Now().UnixNano())
		if err := dbCreateSession(ctx, testClient, userID, orgID, sessionID); err != nil {
			t.Fatalf("dbCreateSession error: %v", err)
		}

		s, err := dbGetSessionRow(ctx, testClient, sessionID)
		if err != nil {
			t.Fatalf("dbGetSessionRow error: %v", err)
		}
		if s.UserID != userID {
			t.Fatalf("expected user ID %d, got %d", userID, s.UserID)
		}
		if s.OrgID != orgID {
			t.Fatalf("expected org ID %d, got %d", orgID, s.OrgID)
		}
		if s.Expires <= time.Now().UnixMilli() {
			t.Fatal("expected session to not be expired")
		}
	})

	t.Run("get not found", func(t *testing.T) {
		_, err := dbGetSessionRow(ctx, testClient, "no-such-session")
		if !errors.Is(err, errNoSession) {
			t.Fatalf("expected errNoSession, got %v", err)
		}
	})

	t.Run("delete user sessions", func(t *testing.T) {
		sessionID := fmt.Sprintf("test-session-del-%d", time.Now().UnixNano())
		if err := dbCreateSession(ctx, testClient, userID, orgID, sessionID); err != nil {
			t.Fatalf("dbCreateSession error: %v", err)
		}

		if err := dbDeleteUserSessions(ctx, testClient, orgID, userID); err != nil {
			t.Fatalf("dbDeleteUserSessions error: %v", err)
		}

		_, err := dbGetSessionRow(ctx, testClient, sessionID)
		if !errors.Is(err, errNoSession) {
			t.Fatalf("expected errNoSession after delete, got %v", err)
		}
	})

}

// --- Cascade Deletes ---

// TestDBCascadeDeleteOrg verifies that deleting an organisation cascades to
// its child users, groups, group bindings, and sessions.
func TestDBCascadeDeleteOrg(t *testing.T) {
	ctx := context.Background()

	orgID, _ := mustCreateOrg(t, uniqueName(t, "org-cascade-org"))
	userID := mustCreateUser(t, orgID, uniqueName(t, "user-cascade-org"))
	groupName := uniqueName(t, "group-cascade-org")
	groupID := mustCreateGroup(t, orgID, groupName)

	if err := dbUpdateUserGroupBindings(ctx, testClient, orgID, userID, []string{groupName}); err != nil {
		t.Fatalf("dbUpdateUserGroupBindings error: %v", err)
	}

	sessionID := fmt.Sprintf("cascade-org-session-%d", time.Now().UnixNano())
	if err := dbCreateSession(ctx, testClient, userID, orgID, sessionID); err != nil {
		t.Fatalf("dbCreateSession error: %v", err)
	}

	if err := dbDeleteOrg(ctx, testClient, orgID); err != nil {
		t.Fatalf("dbDeleteOrg error: %v", err)
	}

	t.Run("user is deleted", func(t *testing.T) {
		_, err := dbGetUser(ctx, testClient, orgID, userID)
		if !errors.Is(err, errNoUser) {
			t.Fatalf("expected errNoUser after org delete, got %v", err)
		}
	})

	t.Run("group is deleted", func(t *testing.T) {
		_, err := dbGetGroup(ctx, testClient, orgID, groupID)
		if !errors.Is(err, errNoGroup) {
			t.Fatalf("expected errNoGroup after org delete, got %v", err)
		}
	})

	t.Run("session is deleted", func(t *testing.T) {
		_, err := dbGetSessionRow(ctx, testClient, sessionID)
		if !errors.Is(err, errNoSession) {
			t.Fatalf("expected errNoSession after org delete, got %v", err)
		}
	})

	t.Run("group bindings are deleted", func(t *testing.T) {
		bindings, err := dbListGroupBindings(ctx, testClient, orgID, userID)
		if err != nil {
			t.Fatalf("dbListGroupBindings error: %v", err)
		}
		if len(bindings) != 0 {
			t.Fatalf("expected 0 bindings after org delete, got %d", len(bindings))
		}
	})
}

// TestDBCascadeDeleteUser verifies that deleting a user cascades to their
// sessions and group bindings.
func TestDBCascadeDeleteUser(t *testing.T) {
	ctx := context.Background()

	orgID, _ := mustCreateOrg(t, uniqueName(t, "org-cascade-user"))
	userID := mustCreateUser(t, orgID, uniqueName(t, "user-cascade-user"))
	groupName := uniqueName(t, "group-cascade-user")
	mustCreateGroup(t, orgID, groupName)

	if err := dbUpdateUserGroupBindings(ctx, testClient, orgID, userID, []string{groupName}); err != nil {
		t.Fatalf("dbUpdateUserGroupBindings error: %v", err)
	}

	sessionID := fmt.Sprintf("cascade-user-session-%d", time.Now().UnixNano())
	if err := dbCreateSession(ctx, testClient, userID, orgID, sessionID); err != nil {
		t.Fatalf("dbCreateSession error: %v", err)
	}

	if err := dbDeleteUser(ctx, testClient, orgID, userID); err != nil {
		t.Fatalf("dbDeleteUser error: %v", err)
	}

	t.Run("session is deleted", func(t *testing.T) {
		_, err := dbGetSessionRow(ctx, testClient, sessionID)
		if !errors.Is(err, errNoSession) {
			t.Fatalf("expected errNoSession after user delete, got %v", err)
		}
	})

	t.Run("group bindings are deleted", func(t *testing.T) {
		bindings, err := dbListGroupBindings(ctx, testClient, orgID, userID)
		if err != nil {
			t.Fatalf("dbListGroupBindings error: %v", err)
		}
		if len(bindings) != 0 {
			t.Fatalf("expected 0 bindings after user delete, got %d", len(bindings))
		}
	})
}

// TestDBCascadeDeleteGroup verifies that deleting a group cascades to its
// group bindings for all member users.
func TestDBCascadeDeleteGroup(t *testing.T) {
	ctx := context.Background()

	orgID, _ := mustCreateOrg(t, uniqueName(t, "org-cascade-group"))
	userID := mustCreateUser(t, orgID, uniqueName(t, "user-cascade-group"))
	groupName := uniqueName(t, "group-cascade-group")
	groupID := mustCreateGroup(t, orgID, groupName)

	if err := dbUpdateUserGroupBindings(ctx, testClient, orgID, userID, []string{groupName}); err != nil {
		t.Fatalf("dbUpdateUserGroupBindings error: %v", err)
	}

	bindings, err := dbListGroupBindings(ctx, testClient, orgID, userID)
	if err != nil {
		t.Fatalf("dbListGroupBindings error: %v", err)
	}
	if len(bindings) != 1 {
		t.Fatalf("expected 1 binding before group delete, got %d", len(bindings))
	}

	if err := dbDeleteGroup(ctx, testClient, orgID, groupID); err != nil {
		t.Fatalf("dbDeleteGroup error: %v", err)
	}

	t.Run("group binding is deleted", func(t *testing.T) {
		bindings, err := dbListGroupBindings(ctx, testClient, orgID, userID)
		if err != nil {
			t.Fatalf("dbListGroupBindings error: %v", err)
		}
		if len(bindings) != 0 {
			t.Fatalf("expected 0 bindings after group delete, got %d", len(bindings))
		}
	})
}
