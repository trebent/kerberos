package integration

import (
	"net/http"
	"slices"
	"testing"

	adminapi "github.com/trebent/kerberos/test/integration/client/admin"
	authbasicapi "github.com/trebent/kerberos/test/integration/client/auth/basic"
)

// TestUserGroupBindingAssign verifies that groups can be assigned to a user and are returned
// by GetUserGroups.
func TestUserGroupBindingAssign(t *testing.T) {
	superLoginResp, err := adminClient.LoginSuperuserWithResponse(
		t.Context(),
		adminapi.LoginSuperuserJSONRequestBody{ClientId: superUserClientID, ClientSecret: superUserClientSecret},
	)
	checkErr(err, t)
	verifyStatusCode(superLoginResp.StatusCode(), http.StatusNoContent, t)
	superRequestEditor := sessionCookieRequestEditor(superLoginResp.HTTPResponse, t)

	orgID, adminRequestEditor := orgWithSession(t, superRequestEditor)

	groupAName := groupName()
	createGroupA, err := basicAuthClient.CreateGroupWithResponse(
		t.Context(),
		orgID,
		authbasicapi.CreateGroupJSONRequestBody{Name: groupAName},
		authbasicapi.RequestEditorFn(adminRequestEditor),
	)
	checkErr(err, t)
	verifyStatusCode(createGroupA.StatusCode(), http.StatusCreated, t)

	groupBName := groupName()
	createGroupB, err := basicAuthClient.CreateGroupWithResponse(
		t.Context(),
		orgID,
		authbasicapi.CreateGroupJSONRequestBody{Name: groupBName},
		authbasicapi.RequestEditorFn(adminRequestEditor),
	)
	checkErr(err, t)
	verifyStatusCode(createGroupB.StatusCode(), http.StatusCreated, t)

	createUserResp, err := basicAuthClient.CreateUserWithResponse(
		t.Context(),
		orgID,
		authbasicapi.CreateUserJSONRequestBody{Name: username(), Password: "password123"},
		authbasicapi.RequestEditorFn(adminRequestEditor),
	)
	checkErr(err, t)
	verifyStatusCode(createUserResp.StatusCode(), http.StatusCreated, t)
	userID := createUserResp.JSON201.Id

	updateResp, err := basicAuthClient.UpdateUserGroupsWithResponse(
		t.Context(),
		orgID,
		userID,
		authbasicapi.UpdateUserGroupsJSONRequestBody{
			{Id: createGroupA.JSON201.Id, Name: groupAName},
			{Id: createGroupB.JSON201.Id, Name: groupBName},
		},
		authbasicapi.RequestEditorFn(adminRequestEditor),
	)
	checkErr(err, t)
	verifyStatusCode(updateResp.StatusCode(), http.StatusOK, t)

	getResp, err := basicAuthClient.GetUserGroupsWithResponse(
		t.Context(),
		orgID,
		userID,
		authbasicapi.RequestEditorFn(adminRequestEditor),
	)
	checkErr(err, t)
	verifyStatusCode(getResp.StatusCode(), http.StatusOK, t)
	groups := *getResp.JSON200
	if !slices.ContainsFunc(groups, func(g authbasicapi.Group) bool { return g.Name == groupAName }) {
		t.Fatalf("expected group %q in user groups, got %v", groupAName, groups)
	}
	if !slices.ContainsFunc(groups, func(g authbasicapi.Group) bool { return g.Name == groupBName }) {
		t.Fatalf("expected group %q in user groups, got %v", groupBName, groups)
	}
}

// TestUserGroupBindingReplace verifies that updating a user's groups replaces the previous
// set entirely — groups removed from the request are no longer returned.
func TestUserGroupBindingReplace(t *testing.T) {
	superLoginResp, err := adminClient.LoginSuperuserWithResponse(
		t.Context(),
		adminapi.LoginSuperuserJSONRequestBody{ClientId: superUserClientID, ClientSecret: superUserClientSecret},
	)
	checkErr(err, t)
	verifyStatusCode(superLoginResp.StatusCode(), http.StatusNoContent, t)
	superRequestEditor := sessionCookieRequestEditor(superLoginResp.HTTPResponse, t)

	orgID, adminRequestEditor := orgWithSession(t, superRequestEditor)

	groupAName := groupName()
	createGroupA, err := basicAuthClient.CreateGroupWithResponse(
		t.Context(),
		orgID,
		authbasicapi.CreateGroupJSONRequestBody{Name: groupAName},
		authbasicapi.RequestEditorFn(adminRequestEditor),
	)
	checkErr(err, t)
	verifyStatusCode(createGroupA.StatusCode(), http.StatusCreated, t)

	groupBName := groupName()
	createGroupB, err := basicAuthClient.CreateGroupWithResponse(
		t.Context(),
		orgID,
		authbasicapi.CreateGroupJSONRequestBody{Name: groupBName},
		authbasicapi.RequestEditorFn(adminRequestEditor),
	)
	checkErr(err, t)
	verifyStatusCode(createGroupB.StatusCode(), http.StatusCreated, t)

	createUserResp, err := basicAuthClient.CreateUserWithResponse(
		t.Context(),
		orgID,
		authbasicapi.CreateUserJSONRequestBody{Name: username(), Password: "password123"},
		authbasicapi.RequestEditorFn(adminRequestEditor),
	)
	checkErr(err, t)
	verifyStatusCode(createUserResp.StatusCode(), http.StatusCreated, t)
	userID := createUserResp.JSON201.Id

	// Assign both groups initially.
	initialUpdateResp, err := basicAuthClient.UpdateUserGroupsWithResponse(
		t.Context(),
		orgID,
		userID,
		authbasicapi.UpdateUserGroupsJSONRequestBody{
			{Id: createGroupA.JSON201.Id, Name: groupAName},
			{Id: createGroupB.JSON201.Id, Name: groupBName},
		},
		authbasicapi.RequestEditorFn(adminRequestEditor),
	)
	checkErr(err, t)
	verifyStatusCode(initialUpdateResp.StatusCode(), http.StatusOK, t)

	// Replace with only group B — group A should be removed.
	replaceResp, err := basicAuthClient.UpdateUserGroupsWithResponse(
		t.Context(),
		orgID,
		userID,
		authbasicapi.UpdateUserGroupsJSONRequestBody{
			{Id: createGroupB.JSON201.Id, Name: groupBName},
		},
		authbasicapi.RequestEditorFn(adminRequestEditor),
	)
	checkErr(err, t)
	verifyStatusCode(replaceResp.StatusCode(), http.StatusOK, t)

	getResp, err := basicAuthClient.GetUserGroupsWithResponse(
		t.Context(),
		orgID,
		userID,
		authbasicapi.RequestEditorFn(adminRequestEditor),
	)
	checkErr(err, t)
	verifyStatusCode(getResp.StatusCode(), http.StatusOK, t)
	groups := *getResp.JSON200
	if slices.ContainsFunc(groups, func(g authbasicapi.Group) bool { return g.Name == groupAName }) {
		t.Fatalf("group %q should have been removed after replace, got %v", groupAName, groups)
	}
	if !slices.ContainsFunc(groups, func(g authbasicapi.Group) bool { return g.Name == groupBName }) {
		t.Fatalf("expected group %q to remain after replace, got %v", groupBName, groups)
	}
}

// TestUserGroupBindingClear verifies that assigning an empty group list removes all group
// memberships from the user.
func TestUserGroupBindingClear(t *testing.T) {
	superLoginResp, err := adminClient.LoginSuperuserWithResponse(
		t.Context(),
		adminapi.LoginSuperuserJSONRequestBody{ClientId: superUserClientID, ClientSecret: superUserClientSecret},
	)
	checkErr(err, t)
	verifyStatusCode(superLoginResp.StatusCode(), http.StatusNoContent, t)
	superRequestEditor := sessionCookieRequestEditor(superLoginResp.HTTPResponse, t)

	orgID, adminRequestEditor := orgWithSession(t, superRequestEditor)

	gName := groupName()
	createGroupResp, err := basicAuthClient.CreateGroupWithResponse(
		t.Context(),
		orgID,
		authbasicapi.CreateGroupJSONRequestBody{Name: gName},
		authbasicapi.RequestEditorFn(adminRequestEditor),
	)
	checkErr(err, t)
	verifyStatusCode(createGroupResp.StatusCode(), http.StatusCreated, t)

	createUserResp, err := basicAuthClient.CreateUserWithResponse(
		t.Context(),
		orgID,
		authbasicapi.CreateUserJSONRequestBody{Name: username(), Password: "password123"},
		authbasicapi.RequestEditorFn(adminRequestEditor),
	)
	checkErr(err, t)
	verifyStatusCode(createUserResp.StatusCode(), http.StatusCreated, t)
	userID := createUserResp.JSON201.Id

	// Assign the group first.
	assignResp, err := basicAuthClient.UpdateUserGroupsWithResponse(
		t.Context(),
		orgID,
		userID,
		authbasicapi.UpdateUserGroupsJSONRequestBody{
			{Id: createGroupResp.JSON201.Id, Name: gName},
		},
		authbasicapi.RequestEditorFn(adminRequestEditor),
	)
	checkErr(err, t)
	verifyStatusCode(assignResp.StatusCode(), http.StatusOK, t)

	// Clear all groups.
	clearResp, err := basicAuthClient.UpdateUserGroupsWithResponse(
		t.Context(),
		orgID,
		userID,
		authbasicapi.UpdateUserGroupsJSONRequestBody{},
		authbasicapi.RequestEditorFn(adminRequestEditor),
	)
	checkErr(err, t)
	verifyStatusCode(clearResp.StatusCode(), http.StatusOK, t)

	getResp, err := basicAuthClient.GetUserGroupsWithResponse(
		t.Context(),
		orgID,
		userID,
		authbasicapi.RequestEditorFn(adminRequestEditor),
	)
	checkErr(err, t)
	verifyStatusCode(getResp.StatusCode(), http.StatusOK, t)
	if len(*getResp.JSON200) != 0 {
		t.Fatalf("expected empty groups after clear, got %v", *getResp.JSON200)
	}
}

// TestUserGroupBindingGet verifies that GetUserGroups returns the expected groups for a user
// that was set up with known group memberships in TestMain.
func TestUserGroupBindingGet(t *testing.T) {
	loginResp, err := adminClient.LoginSuperuserWithResponse(
		t.Context(),
		adminapi.LoginSuperuserJSONRequestBody{ClientId: superUserClientID, ClientSecret: superUserClientSecret},
	)
	checkErr(err, t)
	verifyStatusCode(loginResp.StatusCode(), http.StatusNoContent, t)
	superRequestEditor := sessionCookieRequestEditor(loginResp.HTTPResponse, t)

	getResp, err := basicAuthClient.GetUserGroupsWithResponse(
		t.Context(),
		authbasicapi.Orgid(alwaysOrgID),
		authbasicapi.Userid(alwaysUserID),
		authbasicapi.RequestEditorFn(superRequestEditor),
	)
	checkErr(err, t)
	verifyStatusCode(getResp.StatusCode(), http.StatusOK, t)
	groups := *getResp.JSON200
	for _, expected := range []string{alwaysGroupStaff, alwaysGroupPleb, alwaysGroupDev} {
		if !slices.ContainsFunc(groups, func(g authbasicapi.Group) bool { return g.Name == expected }) {
			t.Fatalf("expected group %q in always-user groups, got %v", expected, groups)
		}
	}
}
