package integration

import (
	lib "github.com/trebent/kerberos/test/lib"
	"net/http"
	"testing"

	adminapi "github.com/trebent/kerberos/test/client/admin"
)

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
