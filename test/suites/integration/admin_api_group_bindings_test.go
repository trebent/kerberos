package integration

import (
	"net/http"
	"testing"

	adminapi "github.com/trebent/kerberos/test/integration/client/admin"
)

// TestAdminUserGroupBindingsAssign verifies that a user can be assigned to groups,
// and that those groups are reflected in the GetUser response.
func TestAdminUserGroupBindingsAssign(t *testing.T) {
	t.Parallel()
	superSession := superLogin(t)

	name := username()
	createUserResp, err := adminClient.CreateUserWithResponse(
		t.Context(),
		adminapi.CreateUserJSONRequestBody{Username: name, Password: "password123"},
		adminapi.RequestEditorFn(requestEditorSessionID(superSession)),
	)
	checkErr(err, t)
	verifyStatusCode(createUserResp.StatusCode(), http.StatusCreated, t)
	userID := mustGetAdminUserID(t, superSession, name)

	grp1Resp, err := adminClient.CreateGroupWithResponse(
		t.Context(),
		adminapi.CreateGroupJSONRequestBody{Name: groupName(), PermissionIDs: allPermissionIDs},
		adminapi.RequestEditorFn(requestEditorSessionID(superSession)),
	)
	checkErr(err, t)
	verifyStatusCode(grp1Resp.StatusCode(), http.StatusCreated, t)
	grp1ID := grp1Resp.JSON201.Id

	grp2Resp, err := adminClient.CreateGroupWithResponse(
		t.Context(),
		adminapi.CreateGroupJSONRequestBody{Name: groupName(), PermissionIDs: allPermissionIDs},
		adminapi.RequestEditorFn(requestEditorSessionID(superSession)),
	)
	checkErr(err, t)
	verifyStatusCode(grp2Resp.StatusCode(), http.StatusCreated, t)
	grp2ID := grp2Resp.JSON201.Id

	updateResp, err := adminClient.UpdateUserGroupsWithResponse(
		t.Context(),
		userID,
		adminapi.UpdateUserGroupsJSONRequestBody{GroupIDs: []int{grp1ID, grp2ID}},
		adminapi.RequestEditorFn(requestEditorSessionID(superSession)),
	)
	checkErr(err, t)
	verifyStatusCode(updateResp.StatusCode(), http.StatusNoContent, t)

	getResp, err := adminClient.GetUserWithResponse(
		t.Context(),
		userID,
		adminapi.RequestEditorFn(requestEditorSessionID(superSession)),
	)
	checkErr(err, t)
	verifyStatusCode(getResp.StatusCode(), http.StatusOK, t)
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
	containsAll([]int{grp1ID, grp2ID}, groupIDs, t)
}

// TestAdminUserGroupBindingsUpdate verifies that a user's group membership can be partially updated
// (groups removed and added).
func TestAdminUserGroupBindingsUpdate(t *testing.T) {
	t.Parallel()
	superSession := superLogin(t)

	name := username()
	createUserResp, err := adminClient.CreateUserWithResponse(
		t.Context(),
		adminapi.CreateUserJSONRequestBody{Username: name, Password: "password123"},
		adminapi.RequestEditorFn(requestEditorSessionID(superSession)),
	)
	checkErr(err, t)
	verifyStatusCode(createUserResp.StatusCode(), http.StatusCreated, t)
	userID := mustGetAdminUserID(t, superSession, name)

	grp1Resp, err := adminClient.CreateGroupWithResponse(
		t.Context(),
		adminapi.CreateGroupJSONRequestBody{Name: groupName(), PermissionIDs: allPermissionIDs},
		adminapi.RequestEditorFn(requestEditorSessionID(superSession)),
	)
	checkErr(err, t)
	verifyStatusCode(grp1Resp.StatusCode(), http.StatusCreated, t)
	grp1ID := grp1Resp.JSON201.Id

	grp2Resp, err := adminClient.CreateGroupWithResponse(
		t.Context(),
		adminapi.CreateGroupJSONRequestBody{Name: groupName(), PermissionIDs: allPermissionIDs},
		adminapi.RequestEditorFn(requestEditorSessionID(superSession)),
	)
	checkErr(err, t)
	verifyStatusCode(grp2Resp.StatusCode(), http.StatusCreated, t)
	grp2ID := grp2Resp.JSON201.Id

	grp3Resp, err := adminClient.CreateGroupWithResponse(
		t.Context(),
		adminapi.CreateGroupJSONRequestBody{Name: groupName(), PermissionIDs: allPermissionIDs},
		adminapi.RequestEditorFn(requestEditorSessionID(superSession)),
	)
	checkErr(err, t)
	verifyStatusCode(grp3Resp.StatusCode(), http.StatusCreated, t)
	grp3ID := grp3Resp.JSON201.Id

	// Assign to grp1 and grp2.
	updateResp, err := adminClient.UpdateUserGroupsWithResponse(
		t.Context(),
		userID,
		adminapi.UpdateUserGroupsJSONRequestBody{GroupIDs: []int{grp1ID, grp2ID}},
		adminapi.RequestEditorFn(requestEditorSessionID(superSession)),
	)
	checkErr(err, t)
	verifyStatusCode(updateResp.StatusCode(), http.StatusNoContent, t)

	// Update: remove grp1, add grp3.
	updateResp, err = adminClient.UpdateUserGroupsWithResponse(
		t.Context(),
		userID,
		adminapi.UpdateUserGroupsJSONRequestBody{GroupIDs: []int{grp2ID, grp3ID}},
		adminapi.RequestEditorFn(requestEditorSessionID(superSession)),
	)
	checkErr(err, t)
	verifyStatusCode(updateResp.StatusCode(), http.StatusNoContent, t)

	getResp, err := adminClient.GetUserWithResponse(
		t.Context(),
		userID,
		adminapi.RequestEditorFn(requestEditorSessionID(superSession)),
	)
	checkErr(err, t)
	verifyStatusCode(getResp.StatusCode(), http.StatusOK, t)
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
	containsAll([]int{grp2ID, grp3ID}, groupIDs, t)
	for _, id := range groupIDs {
		if id == grp1ID {
			t.Fatalf("grp1 should have been removed from user groups")
		}
	}
}

// TestAdminUserGroupBindingsClear verifies that a user's group memberships can be cleared.
func TestAdminUserGroupBindingsClear(t *testing.T) {
	t.Parallel()
	superSession := superLogin(t)

	name := username()
	createUserResp, err := adminClient.CreateUserWithResponse(
		t.Context(),
		adminapi.CreateUserJSONRequestBody{Username: name, Password: "password123"},
		adminapi.RequestEditorFn(requestEditorSessionID(superSession)),
	)
	checkErr(err, t)
	verifyStatusCode(createUserResp.StatusCode(), http.StatusCreated, t)
	userID := mustGetAdminUserID(t, superSession, name)

	grpResp, err := adminClient.CreateGroupWithResponse(
		t.Context(),
		adminapi.CreateGroupJSONRequestBody{Name: groupName(), PermissionIDs: allPermissionIDs},
		adminapi.RequestEditorFn(requestEditorSessionID(superSession)),
	)
	checkErr(err, t)
	verifyStatusCode(grpResp.StatusCode(), http.StatusCreated, t)
	grpID := grpResp.JSON201.Id

	// Assign to the group.
	updateResp, err := adminClient.UpdateUserGroupsWithResponse(
		t.Context(),
		userID,
		adminapi.UpdateUserGroupsJSONRequestBody{GroupIDs: []int{grpID}},
		adminapi.RequestEditorFn(requestEditorSessionID(superSession)),
	)
	checkErr(err, t)
	verifyStatusCode(updateResp.StatusCode(), http.StatusNoContent, t)

	// Clear all groups.
	updateResp, err = adminClient.UpdateUserGroupsWithResponse(
		t.Context(),
		userID,
		adminapi.UpdateUserGroupsJSONRequestBody{GroupIDs: []int{}},
		adminapi.RequestEditorFn(requestEditorSessionID(superSession)),
	)
	checkErr(err, t)
	verifyStatusCode(updateResp.StatusCode(), http.StatusNoContent, t)

	getResp, err := adminClient.GetUserWithResponse(
		t.Context(),
		userID,
		adminapi.RequestEditorFn(requestEditorSessionID(superSession)),
	)
	checkErr(err, t)
	verifyStatusCode(getResp.StatusCode(), http.StatusOK, t)
	if getResp.JSON200.Groups != nil && len(*getResp.JSON200.Groups) != 0 {
		t.Fatalf("expected 0 groups after clear, got %d", len(*getResp.JSON200.Groups))
	}
}

// TestAdminUserGroupBindingsNotFoundUser verifies that updating groups for a non-existent user returns 404.
func TestAdminUserGroupBindingsNotFoundUser(t *testing.T) {
	t.Parallel()
	superSession := superLogin(t)

	updateResp, err := adminClient.UpdateUserGroupsWithResponse(
		t.Context(),
		999999999,
		adminapi.UpdateUserGroupsJSONRequestBody{GroupIDs: []int{}},
		adminapi.RequestEditorFn(requestEditorSessionID(superSession)),
	)
	checkErr(err, t)
	verifyStatusCode(updateResp.StatusCode(), http.StatusNotFound, t)
	verifyAdminAPIErrorResponse(updateResp.JSON404, t)
}
