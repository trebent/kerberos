package integration

import (
	lib "github.com/trebent/kerberos/test/lib"
	"net/http"
	"testing"

	adminapi "github.com/trebent/kerberos/test/client/admin"
)

// TestAdminUserCreate verifies that a new admin user can be created via a superuser session.
func TestAdminUserCreate(t *testing.T) {
	t.Parallel()
	superRequestEditor := lib.SuperLogin(t)

	createResp, err := lib.AdminClient.CreateUserWithResponse(
		t.Context(),
		adminapi.CreateUserJSONRequestBody{Username: lib.Username(), Password: "password123"},
		adminapi.RequestEditorFn(superRequestEditor),
	)
	lib.CheckErr(err, t)
	lib.VerifyStatusCode(createResp.StatusCode(), http.StatusCreated, t)
}

// TestAdminUserCreateConflict verifies that creating a duplicate admin username is rejected.
func TestAdminUserCreateConflict(t *testing.T) {
	t.Parallel()
	superRequestEditor := lib.SuperLogin(t)

	name := lib.Username()
	createResp, err := lib.AdminClient.CreateUserWithResponse(
		t.Context(),
		adminapi.CreateUserJSONRequestBody{Username: name, Password: "password123"},
		adminapi.RequestEditorFn(superRequestEditor),
	)
	lib.CheckErr(err, t)
	lib.VerifyStatusCode(createResp.StatusCode(), http.StatusCreated, t)

	dupResp, err := lib.AdminClient.CreateUserWithResponse(
		t.Context(),
		adminapi.CreateUserJSONRequestBody{Username: name, Password: "other-password"},
		adminapi.RequestEditorFn(superRequestEditor),
	)
	lib.CheckErr(err, t)
	lib.VerifyStatusCode(dupResp.StatusCode(), http.StatusConflict, t)
	lib.VerifyAdminAPIErrorResponse(dupResp.JSON409, t)
}

// TestAdminUserList verifies that a newly created admin user appears in the list response.
func TestAdminUserList(t *testing.T) {
	t.Parallel()
	superRequestEditor := lib.SuperLogin(t)

	name := lib.Username()
	createResp, err := lib.AdminClient.CreateUserWithResponse(
		t.Context(),
		adminapi.CreateUserJSONRequestBody{Username: name, Password: "password123"},
		adminapi.RequestEditorFn(superRequestEditor),
	)
	lib.CheckErr(err, t)
	lib.VerifyStatusCode(createResp.StatusCode(), http.StatusCreated, t)

	listResp, err := lib.AdminClient.GetUsersWithResponse(
		t.Context(),
		adminapi.RequestEditorFn(superRequestEditor),
	)
	lib.CheckErr(err, t)
	lib.VerifyStatusCode(listResp.StatusCode(), http.StatusOK, t)
	for _, u := range *listResp.JSON200 {
		if u.Username == name {
			return
		}
	}
	t.Fatalf("admin user %q not found in list", name)
}

// TestAdminUserGet verifies that a created admin user can be fetched by ID.
func TestAdminUserGet(t *testing.T) {
	t.Parallel()
	superRequestEditor := lib.SuperLogin(t)

	name := lib.Username()
	createResp, err := lib.AdminClient.CreateUserWithResponse(
		t.Context(),
		adminapi.CreateUserJSONRequestBody{Username: name, Password: "password123"},
		adminapi.RequestEditorFn(superRequestEditor),
	)
	lib.CheckErr(err, t)
	lib.VerifyStatusCode(createResp.StatusCode(), http.StatusCreated, t)

	getResp, err := lib.AdminClient.GetUserWithResponse(
		t.Context(),
		createResp.JSON201.Id,
		adminapi.RequestEditorFn(superRequestEditor),
	)
	lib.CheckErr(err, t)
	lib.VerifyStatusCode(getResp.StatusCode(), http.StatusOK, t)
	lib.Matches(getResp.JSON200.Username, name, t)
	lib.Matches(getResp.JSON200.Id, createResp.JSON201.Id, t)
}

// TestAdminUserGetNotFound verifies that fetching a non-existent admin user returns 404.
func TestAdminUserGetNotFound(t *testing.T) {
	t.Parallel()
	superRequestEditor := lib.SuperLogin(t)

	getResp, err := lib.AdminClient.GetUserWithResponse(
		t.Context(),
		999999999,
		adminapi.RequestEditorFn(superRequestEditor),
	)
	lib.CheckErr(err, t)
	lib.VerifyStatusCode(getResp.StatusCode(), http.StatusNotFound, t)
	lib.VerifyAdminAPIErrorResponse(getResp.JSON404, t)
}

// TestAdminUserUpdate verifies that an admin user's username can be updated.
func TestAdminUserUpdate(t *testing.T) {
	t.Parallel()
	superRequestEditor := lib.SuperLogin(t)

	name := lib.Username()
	createResp, err := lib.AdminClient.CreateUserWithResponse(
		t.Context(),
		adminapi.CreateUserJSONRequestBody{Username: name, Password: "password123"},
		adminapi.RequestEditorFn(superRequestEditor),
	)
	lib.CheckErr(err, t)
	lib.VerifyStatusCode(createResp.StatusCode(), http.StatusCreated, t)

	newName := lib.Username()
	updateResp, err := lib.AdminClient.UpdateUserWithResponse(
		t.Context(),
		createResp.JSON201.Id,
		adminapi.UpdateUserJSONRequestBody{Username: newName},
		adminapi.RequestEditorFn(superRequestEditor),
	)
	lib.CheckErr(err, t)
	lib.VerifyStatusCode(updateResp.StatusCode(), http.StatusNoContent, t)

	getResp, err := lib.AdminClient.GetUserWithResponse(
		t.Context(),
		createResp.JSON201.Id,
		adminapi.RequestEditorFn(superRequestEditor),
	)
	lib.CheckErr(err, t)
	lib.VerifyStatusCode(getResp.StatusCode(), http.StatusOK, t)
	lib.Matches(getResp.JSON200.Username, newName, t)
}

// TestAdminUserUpdateConflict verifies that updating an admin user's username to an existing username returns a conflict.
func TestAdminUserUpdateConflict(t *testing.T) {
	t.Parallel()
	superRequestEditor := lib.SuperLogin(t)

	name := lib.Username()
	createResp, err := lib.AdminClient.CreateUserWithResponse(
		t.Context(),
		adminapi.CreateUserJSONRequestBody{Username: name, Password: "password123"},
		adminapi.RequestEditorFn(superRequestEditor),
	)
	lib.CheckErr(err, t)
	lib.VerifyStatusCode(createResp.StatusCode(), http.StatusCreated, t)

	name2 := lib.Username()
	createResp2, err := lib.AdminClient.CreateUserWithResponse(
		t.Context(),
		adminapi.CreateUserJSONRequestBody{Username: name2, Password: "password123"},
		adminapi.RequestEditorFn(superRequestEditor),
	)
	lib.CheckErr(err, t)
	lib.VerifyStatusCode(createResp2.StatusCode(), http.StatusCreated, t)

	updateResp, err := lib.AdminClient.UpdateUserWithResponse(
		t.Context(),
		createResp.JSON201.Id,
		adminapi.UpdateUserJSONRequestBody{Username: name2},
		adminapi.RequestEditorFn(superRequestEditor),
	)
	lib.CheckErr(err, t)
	lib.VerifyStatusCode(updateResp.StatusCode(), http.StatusConflict, t)
}

// TestAdminUserDelete verifies that an admin user can be deleted and is no longer retrievable.
func TestAdminUserDelete(t *testing.T) {
	t.Parallel()
	superRequestEditor := lib.SuperLogin(t)

	name := lib.Username()
	createResp, err := lib.AdminClient.CreateUserWithResponse(
		t.Context(),
		adminapi.CreateUserJSONRequestBody{Username: name, Password: "password123"},
		adminapi.RequestEditorFn(superRequestEditor),
	)
	lib.CheckErr(err, t)
	lib.VerifyStatusCode(createResp.StatusCode(), http.StatusCreated, t)

	deleteResp, err := lib.AdminClient.DeleteUserWithResponse(
		t.Context(),
		createResp.JSON201.Id,
		adminapi.RequestEditorFn(superRequestEditor),
	)
	lib.CheckErr(err, t)
	lib.VerifyStatusCode(deleteResp.StatusCode(), http.StatusNoContent, t)

	getResp, err := lib.AdminClient.GetUserWithResponse(
		t.Context(),
		createResp.JSON201.Id,
		adminapi.RequestEditorFn(superRequestEditor),
	)
	lib.CheckErr(err, t)
	lib.VerifyStatusCode(getResp.StatusCode(), http.StatusNotFound, t)
}

// TestAdminUserDeleteNotFound verifies that deleting a non-existent admin user returns 404.
func TestAdminUserDeleteNotFound(t *testing.T) {
	t.Parallel()
	superRequestEditor := lib.SuperLogin(t)

	deleteResp, err := lib.AdminClient.DeleteUserWithResponse(
		t.Context(),
		999999999,
		adminapi.RequestEditorFn(superRequestEditor),
	)
	lib.CheckErr(err, t)
	lib.VerifyStatusCode(deleteResp.StatusCode(), http.StatusNotFound, t)
	lib.VerifyAdminAPIErrorResponse(deleteResp.JSON404, t)
}

// TestAdminUserGroupBindingsAssign verifies that a user can be assigned to groups,
// and that those groups are reflected in the GetUser response.
func TestAdminUserGroupBindingsAssign(t *testing.T) {
	t.Parallel()
	superRequestEditor := lib.SuperLogin(t)

	name := lib.Username()
	createUserResp, err := lib.AdminClient.CreateUserWithResponse(
		t.Context(),
		adminapi.CreateUserJSONRequestBody{Username: name, Password: "password123"},
		adminapi.RequestEditorFn(superRequestEditor),
	)
	lib.CheckErr(err, t)
	lib.VerifyStatusCode(createUserResp.StatusCode(), http.StatusCreated, t)
	userID := lib.MustGetAdminUserID(t, superRequestEditor, name)

	grp1Resp, err := lib.AdminClient.CreateGroupWithResponse(
		t.Context(),
		adminapi.CreateGroupJSONRequestBody{Name: lib.GroupName(), PermissionIDs: allPermissionIDs},
		adminapi.RequestEditorFn(superRequestEditor),
	)
	lib.CheckErr(err, t)
	lib.VerifyStatusCode(grp1Resp.StatusCode(), http.StatusCreated, t)
	grp1ID := grp1Resp.JSON201.Id

	grp2Resp, err := lib.AdminClient.CreateGroupWithResponse(
		t.Context(),
		adminapi.CreateGroupJSONRequestBody{Name: lib.GroupName(), PermissionIDs: allPermissionIDs},
		adminapi.RequestEditorFn(superRequestEditor),
	)
	lib.CheckErr(err, t)
	lib.VerifyStatusCode(grp2Resp.StatusCode(), http.StatusCreated, t)
	grp2ID := grp2Resp.JSON201.Id

	updateResp, err := lib.AdminClient.UpdateUserGroupsWithResponse(
		t.Context(),
		userID,
		adminapi.UpdateUserGroupsJSONRequestBody{GroupIDs: []int{grp1ID, grp2ID}},
		adminapi.RequestEditorFn(superRequestEditor),
	)
	lib.CheckErr(err, t)
	lib.VerifyStatusCode(updateResp.StatusCode(), http.StatusNoContent, t)

	getResp, err := lib.AdminClient.GetUserWithResponse(
		t.Context(),
		userID,
		adminapi.RequestEditorFn(superRequestEditor),
	)
	lib.CheckErr(err, t)
	lib.VerifyStatusCode(getResp.StatusCode(), http.StatusOK, t)
	if getResp.JSON200.Groups == nil {
		t.Fatal("expected non-nil groups on user")
	}
	if len(*getResp.JSON200.Groups) != 2 {
		t.Fatalf("expected 2 groups, got %d", len(*getResp.JSON200.Groups))
	}
	groupIDs := make([]int, 0, len(*getResp.JSON200.Groups))
	for _, g := range *getResp.JSON200.Groups {
		groupIDs = append(groupIDs, g.Id)
	}
	lib.ContainsAll([]int{grp1ID, grp2ID}, groupIDs, t)
}

// TestAdminUserGroupBindingsUpdate verifies that a user's group membership can be partially updated
// (groups removed and added).
func TestAdminUserGroupBindingsUpdate(t *testing.T) {
	t.Parallel()
	superRequestEditor := lib.SuperLogin(t)

	name := lib.Username()
	createUserResp, err := lib.AdminClient.CreateUserWithResponse(
		t.Context(),
		adminapi.CreateUserJSONRequestBody{Username: name, Password: "password123"},
		adminapi.RequestEditorFn(superRequestEditor),
	)
	lib.CheckErr(err, t)
	lib.VerifyStatusCode(createUserResp.StatusCode(), http.StatusCreated, t)
	userID := lib.MustGetAdminUserID(t, superRequestEditor, name)

	grp1Resp, err := lib.AdminClient.CreateGroupWithResponse(
		t.Context(),
		adminapi.CreateGroupJSONRequestBody{Name: lib.GroupName(), PermissionIDs: allPermissionIDs},
		adminapi.RequestEditorFn(superRequestEditor),
	)
	lib.CheckErr(err, t)
	lib.VerifyStatusCode(grp1Resp.StatusCode(), http.StatusCreated, t)
	grp1ID := grp1Resp.JSON201.Id

	grp2Resp, err := lib.AdminClient.CreateGroupWithResponse(
		t.Context(),
		adminapi.CreateGroupJSONRequestBody{Name: lib.GroupName(), PermissionIDs: allPermissionIDs},
		adminapi.RequestEditorFn(superRequestEditor),
	)
	lib.CheckErr(err, t)
	lib.VerifyStatusCode(grp2Resp.StatusCode(), http.StatusCreated, t)
	grp2ID := grp2Resp.JSON201.Id

	grp3Resp, err := lib.AdminClient.CreateGroupWithResponse(
		t.Context(),
		adminapi.CreateGroupJSONRequestBody{Name: lib.GroupName(), PermissionIDs: allPermissionIDs},
		adminapi.RequestEditorFn(superRequestEditor),
	)
	lib.CheckErr(err, t)
	lib.VerifyStatusCode(grp3Resp.StatusCode(), http.StatusCreated, t)
	grp3ID := grp3Resp.JSON201.Id

	// Assign to grp1 and grp2.
	updateResp, err := lib.AdminClient.UpdateUserGroupsWithResponse(
		t.Context(),
		userID,
		adminapi.UpdateUserGroupsJSONRequestBody{GroupIDs: []int{grp1ID, grp2ID}},
		adminapi.RequestEditorFn(superRequestEditor),
	)
	lib.CheckErr(err, t)
	lib.VerifyStatusCode(updateResp.StatusCode(), http.StatusNoContent, t)

	// Update: remove grp1, add grp3.
	updateResp, err = lib.AdminClient.UpdateUserGroupsWithResponse(
		t.Context(),
		userID,
		adminapi.UpdateUserGroupsJSONRequestBody{GroupIDs: []int{grp2ID, grp3ID}},
		adminapi.RequestEditorFn(superRequestEditor),
	)
	lib.CheckErr(err, t)
	lib.VerifyStatusCode(updateResp.StatusCode(), http.StatusNoContent, t)

	getResp, err := lib.AdminClient.GetUserWithResponse(
		t.Context(),
		userID,
		adminapi.RequestEditorFn(superRequestEditor),
	)
	lib.CheckErr(err, t)
	lib.VerifyStatusCode(getResp.StatusCode(), http.StatusOK, t)
	if getResp.JSON200.Groups == nil {
		t.Fatal("expected non-nil groups on user")
	}
	if len(*getResp.JSON200.Groups) != 2 {
		t.Fatalf("expected 2 groups after update, got %d", len(*getResp.JSON200.Groups))
	}
	groupIDs := make([]int, 0, len(*getResp.JSON200.Groups))
	for _, g := range *getResp.JSON200.Groups {
		groupIDs = append(groupIDs, g.Id)
	}
	lib.ContainsAll([]int{grp2ID, grp3ID}, groupIDs, t)
	for _, id := range groupIDs {
		if id == grp1ID {
			t.Fatalf("grp1 should have been removed from user groups")
		}
	}
}

// TestAdminUserGroupBindingsClear verifies that a user's group memberships can be cleared.
func TestAdminUserGroupBindingsClear(t *testing.T) {
	t.Parallel()
	superRequestEditor := lib.SuperLogin(t)

	name := lib.Username()
	createUserResp, err := lib.AdminClient.CreateUserWithResponse(
		t.Context(),
		adminapi.CreateUserJSONRequestBody{Username: name, Password: "password123"},
		adminapi.RequestEditorFn(superRequestEditor),
	)
	lib.CheckErr(err, t)
	lib.VerifyStatusCode(createUserResp.StatusCode(), http.StatusCreated, t)
	userID := lib.MustGetAdminUserID(t, superRequestEditor, name)

	grpResp, err := lib.AdminClient.CreateGroupWithResponse(
		t.Context(),
		adminapi.CreateGroupJSONRequestBody{Name: lib.GroupName(), PermissionIDs: allPermissionIDs},
		adminapi.RequestEditorFn(superRequestEditor),
	)
	lib.CheckErr(err, t)
	lib.VerifyStatusCode(grpResp.StatusCode(), http.StatusCreated, t)
	grpID := grpResp.JSON201.Id

	// Assign to the group.
	updateResp, err := lib.AdminClient.UpdateUserGroupsWithResponse(
		t.Context(),
		userID,
		adminapi.UpdateUserGroupsJSONRequestBody{GroupIDs: []int{grpID}},
		adminapi.RequestEditorFn(superRequestEditor),
	)
	lib.CheckErr(err, t)
	lib.VerifyStatusCode(updateResp.StatusCode(), http.StatusNoContent, t)

	// Clear all groups.
	updateResp, err = lib.AdminClient.UpdateUserGroupsWithResponse(
		t.Context(),
		userID,
		adminapi.UpdateUserGroupsJSONRequestBody{GroupIDs: []int{}},
		adminapi.RequestEditorFn(superRequestEditor),
	)
	lib.CheckErr(err, t)
	lib.VerifyStatusCode(updateResp.StatusCode(), http.StatusNoContent, t)

	getResp, err := lib.AdminClient.GetUserWithResponse(
		t.Context(),
		userID,
		adminapi.RequestEditorFn(superRequestEditor),
	)
	lib.CheckErr(err, t)
	lib.VerifyStatusCode(getResp.StatusCode(), http.StatusOK, t)
	if getResp.JSON200.Groups != nil && len(*getResp.JSON200.Groups) != 0 {
		t.Fatalf("expected 0 groups after clear, got %d", len(*getResp.JSON200.Groups))
	}
}

// TestAdminUserGroupBindingsNotFoundUser verifies that updating groups for a non-existent user returns 404.
func TestAdminUserGroupBindingsNotFoundUser(t *testing.T) {
	t.Parallel()
	superRequestEditor := lib.SuperLogin(t)

	updateResp, err := lib.AdminClient.UpdateUserGroupsWithResponse(
		t.Context(),
		999999999,
		adminapi.UpdateUserGroupsJSONRequestBody{GroupIDs: []int{}},
		adminapi.RequestEditorFn(superRequestEditor),
	)
	lib.CheckErr(err, t)
	lib.VerifyStatusCode(updateResp.StatusCode(), http.StatusNotFound, t)
	lib.VerifyAdminAPIErrorResponse(updateResp.JSON404, t)
}
