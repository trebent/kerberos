package integration

import (
	lib "github.com/trebent/kerberos/test/lib"
	"net/http"
	"slices"
	"testing"

	adminapi "github.com/trebent/kerberos/test/client/admin"
	authbasicapi "github.com/trebent/kerberos/test/client/auth/basic"
)

// TestUserGroupBindingAssign verifies that groups can be assigned to a user and are returned
// by GetUserGroups.
func TestUserGroupBindingAssign(t *testing.T) {
	superLoginResp, err := lib.AdminClient.LoginSuperuserWithResponse(
		t.Context(),
		adminapi.LoginSuperuserJSONRequestBody{ClientId: lib.SuperUserClientID, ClientSecret: lib.SuperUserClientSecret},
	)
	lib.CheckErr(err, t)
	lib.VerifyStatusCode(superLoginResp.StatusCode(), http.StatusNoContent, t)
	superRequestEditor := lib.SessionCookieRequestEditor(superLoginResp.HTTPResponse, t)

	orgID, adminRequestEditor := lib.OrgWithSession(t, superRequestEditor)

	groupAName := lib.GroupName()
	createGroupA, err := lib.BasicAuthClient.CreateGroupWithResponse(
		t.Context(),
		orgID,
		authbasicapi.CreateGroupJSONRequestBody{Name: groupAName},
		authbasicapi.RequestEditorFn(adminRequestEditor),
	)
	lib.CheckErr(err, t)
	lib.VerifyStatusCode(createGroupA.StatusCode(), http.StatusCreated, t)

	groupBName := lib.GroupName()
	createGroupB, err := lib.BasicAuthClient.CreateGroupWithResponse(
		t.Context(),
		orgID,
		authbasicapi.CreateGroupJSONRequestBody{Name: groupBName},
		authbasicapi.RequestEditorFn(adminRequestEditor),
	)
	lib.CheckErr(err, t)
	lib.VerifyStatusCode(createGroupB.StatusCode(), http.StatusCreated, t)

	createUserResp, err := lib.BasicAuthClient.CreateUserWithResponse(
		t.Context(),
		orgID,
		authbasicapi.CreateUserJSONRequestBody{Name: lib.Username(), Password: "password123"},
		authbasicapi.RequestEditorFn(adminRequestEditor),
	)
	lib.CheckErr(err, t)
	lib.VerifyStatusCode(createUserResp.StatusCode(), http.StatusCreated, t)
	userID := createUserResp.JSON201.Id

	updateResp, err := lib.BasicAuthClient.UpdateUserGroupsWithResponse(
		t.Context(),
		orgID,
		userID,
		authbasicapi.UpdateUserGroupsJSONRequestBody{
			{Id: createGroupA.JSON201.Id, Name: groupAName},
			{Id: createGroupB.JSON201.Id, Name: groupBName},
		},
		authbasicapi.RequestEditorFn(adminRequestEditor),
	)
	lib.CheckErr(err, t)
	lib.VerifyStatusCode(updateResp.StatusCode(), http.StatusOK, t)

	getResp, err := lib.BasicAuthClient.GetUserGroupsWithResponse(
		t.Context(),
		orgID,
		userID,
		authbasicapi.RequestEditorFn(adminRequestEditor),
	)
	lib.CheckErr(err, t)
	lib.VerifyStatusCode(getResp.StatusCode(), http.StatusOK, t)
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
	superLoginResp, err := lib.AdminClient.LoginSuperuserWithResponse(
		t.Context(),
		adminapi.LoginSuperuserJSONRequestBody{ClientId: lib.SuperUserClientID, ClientSecret: lib.SuperUserClientSecret},
	)
	lib.CheckErr(err, t)
	lib.VerifyStatusCode(superLoginResp.StatusCode(), http.StatusNoContent, t)
	superRequestEditor := lib.SessionCookieRequestEditor(superLoginResp.HTTPResponse, t)

	orgID, adminRequestEditor := lib.OrgWithSession(t, superRequestEditor)

	groupAName := lib.GroupName()
	createGroupA, err := lib.BasicAuthClient.CreateGroupWithResponse(
		t.Context(),
		orgID,
		authbasicapi.CreateGroupJSONRequestBody{Name: groupAName},
		authbasicapi.RequestEditorFn(adminRequestEditor),
	)
	lib.CheckErr(err, t)
	lib.VerifyStatusCode(createGroupA.StatusCode(), http.StatusCreated, t)

	groupBName := lib.GroupName()
	createGroupB, err := lib.BasicAuthClient.CreateGroupWithResponse(
		t.Context(),
		orgID,
		authbasicapi.CreateGroupJSONRequestBody{Name: groupBName},
		authbasicapi.RequestEditorFn(adminRequestEditor),
	)
	lib.CheckErr(err, t)
	lib.VerifyStatusCode(createGroupB.StatusCode(), http.StatusCreated, t)

	createUserResp, err := lib.BasicAuthClient.CreateUserWithResponse(
		t.Context(),
		orgID,
		authbasicapi.CreateUserJSONRequestBody{Name: lib.Username(), Password: "password123"},
		authbasicapi.RequestEditorFn(adminRequestEditor),
	)
	lib.CheckErr(err, t)
	lib.VerifyStatusCode(createUserResp.StatusCode(), http.StatusCreated, t)
	userID := createUserResp.JSON201.Id

	// Assign both groups initially.
	initialUpdateResp, err := lib.BasicAuthClient.UpdateUserGroupsWithResponse(
		t.Context(),
		orgID,
		userID,
		authbasicapi.UpdateUserGroupsJSONRequestBody{
			{Id: createGroupA.JSON201.Id, Name: groupAName},
			{Id: createGroupB.JSON201.Id, Name: groupBName},
		},
		authbasicapi.RequestEditorFn(adminRequestEditor),
	)
	lib.CheckErr(err, t)
	lib.VerifyStatusCode(initialUpdateResp.StatusCode(), http.StatusOK, t)

	// Replace with only group B — group A should be removed.
	replaceResp, err := lib.BasicAuthClient.UpdateUserGroupsWithResponse(
		t.Context(),
		orgID,
		userID,
		authbasicapi.UpdateUserGroupsJSONRequestBody{
			{Id: createGroupB.JSON201.Id, Name: groupBName},
		},
		authbasicapi.RequestEditorFn(adminRequestEditor),
	)
	lib.CheckErr(err, t)
	lib.VerifyStatusCode(replaceResp.StatusCode(), http.StatusOK, t)

	getResp, err := lib.BasicAuthClient.GetUserGroupsWithResponse(
		t.Context(),
		orgID,
		userID,
		authbasicapi.RequestEditorFn(adminRequestEditor),
	)
	lib.CheckErr(err, t)
	lib.VerifyStatusCode(getResp.StatusCode(), http.StatusOK, t)
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
	superLoginResp, err := lib.AdminClient.LoginSuperuserWithResponse(
		t.Context(),
		adminapi.LoginSuperuserJSONRequestBody{ClientId: lib.SuperUserClientID, ClientSecret: lib.SuperUserClientSecret},
	)
	lib.CheckErr(err, t)
	lib.VerifyStatusCode(superLoginResp.StatusCode(), http.StatusNoContent, t)
	superRequestEditor := lib.SessionCookieRequestEditor(superLoginResp.HTTPResponse, t)

	orgID, adminRequestEditor := lib.OrgWithSession(t, superRequestEditor)

	gName := lib.GroupName()
	createGroupResp, err := lib.BasicAuthClient.CreateGroupWithResponse(
		t.Context(),
		orgID,
		authbasicapi.CreateGroupJSONRequestBody{Name: gName},
		authbasicapi.RequestEditorFn(adminRequestEditor),
	)
	lib.CheckErr(err, t)
	lib.VerifyStatusCode(createGroupResp.StatusCode(), http.StatusCreated, t)

	createUserResp, err := lib.BasicAuthClient.CreateUserWithResponse(
		t.Context(),
		orgID,
		authbasicapi.CreateUserJSONRequestBody{Name: lib.Username(), Password: "password123"},
		authbasicapi.RequestEditorFn(adminRequestEditor),
	)
	lib.CheckErr(err, t)
	lib.VerifyStatusCode(createUserResp.StatusCode(), http.StatusCreated, t)
	userID := createUserResp.JSON201.Id

	// Assign the group first.
	assignResp, err := lib.BasicAuthClient.UpdateUserGroupsWithResponse(
		t.Context(),
		orgID,
		userID,
		authbasicapi.UpdateUserGroupsJSONRequestBody{
			{Id: createGroupResp.JSON201.Id, Name: gName},
		},
		authbasicapi.RequestEditorFn(adminRequestEditor),
	)
	lib.CheckErr(err, t)
	lib.VerifyStatusCode(assignResp.StatusCode(), http.StatusOK, t)

	// Clear all groups.
	clearResp, err := lib.BasicAuthClient.UpdateUserGroupsWithResponse(
		t.Context(),
		orgID,
		userID,
		authbasicapi.UpdateUserGroupsJSONRequestBody{},
		authbasicapi.RequestEditorFn(adminRequestEditor),
	)
	lib.CheckErr(err, t)
	lib.VerifyStatusCode(clearResp.StatusCode(), http.StatusOK, t)

	getResp, err := lib.BasicAuthClient.GetUserGroupsWithResponse(
		t.Context(),
		orgID,
		userID,
		authbasicapi.RequestEditorFn(adminRequestEditor),
	)
	lib.CheckErr(err, t)
	lib.VerifyStatusCode(getResp.StatusCode(), http.StatusOK, t)
	if len(*getResp.JSON200) != 0 {
		t.Fatalf("expected empty groups after clear, got %v", *getResp.JSON200)
	}
}

// TestUserGroupBindingGet verifies that GetUserGroups returns the expected groups for a user
// that was set up with known group memberships in TestMain.
func TestUserGroupBindingGet(t *testing.T) {
	loginResp, err := lib.AdminClient.LoginSuperuserWithResponse(
		t.Context(),
		adminapi.LoginSuperuserJSONRequestBody{ClientId: lib.SuperUserClientID, ClientSecret: lib.SuperUserClientSecret},
	)
	lib.CheckErr(err, t)
	lib.VerifyStatusCode(loginResp.StatusCode(), http.StatusNoContent, t)
	superRequestEditor := lib.SessionCookieRequestEditor(loginResp.HTTPResponse, t)

	getResp, err := lib.BasicAuthClient.GetUserGroupsWithResponse(
		t.Context(),
		authbasicapi.Orgid(alwaysOrgID),
		authbasicapi.Userid(alwaysUserID),
		authbasicapi.RequestEditorFn(superRequestEditor),
	)
	lib.CheckErr(err, t)
	lib.VerifyStatusCode(getResp.StatusCode(), http.StatusOK, t)
	groups := *getResp.JSON200
	for _, expected := range []string{alwaysGroupStaff, alwaysGroupPleb, alwaysGroupDev} {
		if !slices.ContainsFunc(groups, func(g authbasicapi.Group) bool { return g.Name == expected }) {
			t.Fatalf("expected group %q in always-user groups, got %v", expected, groups)
		}
	}
}
