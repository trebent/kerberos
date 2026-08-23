package integration

import (
	lib "github.com/trebent/kerberos/test/lib"
	"net/http"
	"testing"

	adminapi "github.com/trebent/kerberos/test/client/admin"
)

// allPermissionIDs is the base set of all available admin group permissions.
// Tests that create admin groups should include these to avoid breaking permission-gated endpoints.
var allPermissionIDs = []int{1, 2, 3, 4, 5, 6, 7}

func TestAdminLoginSuperuser(t *testing.T) {
	superRequestEditor := lib.SuperLogin(t)

	t.Log("Logging the superuser out")
	superLogoutResp, err := lib.AdminClient.LogoutSuperuserWithResponse(
		t.Context(),
		adminapi.RequestEditorFn(superRequestEditor),
	)
	lib.CheckErr(err, t)
	lib.VerifyStatusCode(superLogoutResp.StatusCode(), http.StatusNoContent, t)

	t.Log("Running a GET flow request with the old session to verify it is invalidated")
	// Verify the old session is truly invalidated by attempting to access a protected endpoint with it.
	getFlowResp, err := lib.AdminClient.GetFlowWithResponse(
		t.Context(),
		adminapi.RequestEditorFn(superRequestEditor),
	)
	lib.CheckErr(err, t)
	lib.VerifyStatusCode(getFlowResp.StatusCode(), http.StatusUnauthorized, t)
}

func TestAdminLoginSuperuserFailure(t *testing.T) {
	t.Parallel()
	superLoginResp, err := lib.AdminClient.LoginSuperuserWithResponse(
		t.Context(),
		adminapi.LoginSuperuserJSONRequestBody{ClientId: lib.SuperUserClientID, ClientSecret: "not-correct"},
	)
	lib.CheckErr(err, t)
	lib.VerifyStatusCode(superLoginResp.StatusCode(), http.StatusUnauthorized, t)
	lib.VerifyAdminAPIErrorResponse(superLoginResp.JSON401, t)
}

func TestAdminOASFailure(t *testing.T) {
	t.Parallel()
	badSuperLoginResp, err := lib.AdminClient.LoginSuperuserWithResponse(t.Context(), adminapi.LoginSuperuserJSONRequestBody{})
	lib.CheckErr(err, t)
	lib.VerifyStatusCode(badSuperLoginResp.StatusCode(), http.StatusBadRequest, t)
	lib.VerifyAdminAPIErrorResponse(badSuperLoginResp.JSON400, t)
}

func TestAdminGetFlow(t *testing.T) {
	t.Parallel()
	superRequestEditor := lib.SuperLogin(t)

	getFlowResp, err := lib.AdminClient.GetFlowWithResponse(
		t.Context(),
		adminapi.RequestEditorFn(superRequestEditor),
	)
	lib.CheckErr(err, t)
	lib.VerifyStatusCode(getFlowResp.StatusCode(), http.StatusOK, t)

	for i, component := range *getFlowResp.JSON200 {
		t.Logf("Flow component index: %d name: %s", i, component.Name)

		switch component.Name {
		case "obs":
			if i != 0 {
				t.Error("observability component should have index 0")
			}
			_, err := component.Data.AsFlowMetaDataObservability()
			if err != nil {
				t.Fatalf("Failed to parse observability component data: %v", err)
			}
		case "router":
			if i != 1 {
				t.Error("router component should have index 1")
			}
			_, err := component.Data.AsFlowMetaDataRouter()
			if err != nil {
				t.Fatalf("Failed to parse router component data: %v", err)
			}
		case "authorizer":
			if i != 2 {
				t.Error("authorizer component should have index 2")
			}
			_, err := component.Data.AsFlowMetaDataAuth()
			if err != nil {
				t.Fatalf("Failed to parse authorizer component data: %v", err)
			}
		case "oas-validator":
			if i != 3 {
				t.Error("oas-validator component should have index 3")
			}
			_, err := component.Data.AsFlowMetaDataOAS()
			if err != nil {
				t.Fatalf("Failed to parse oas-validator component data: %v", err)
			}
		case "forwarder":
			if i != 4 {
				t.Error("forwarder component should have index 4")
			}
			_, err := component.Data.AsNoFlowMetaData()
			if err != nil {
				t.Fatalf("Failed to parse forwarder component data: %v", err)
			}
		default:
			t.Errorf("Unexpected flow component name: %s", component.Name)
		}
	}
}

func TestAdminGetBackendOASNotFound(t *testing.T) {
	t.Parallel()
	superRequestEditor := lib.SuperLogin(t)

	resp, err := lib.AdminClient.GetBackendOASWithResponse(
		t.Context(),
		"nonexistent-backend",
		adminapi.RequestEditorFn(superRequestEditor),
	)
	lib.CheckErr(err, t)
	lib.VerifyStatusCode(resp.StatusCode(), http.StatusNotFound, t)
	lib.VerifyAdminAPIErrorResponse(resp.JSON404, t)
}

func TestAdminGetBackendOAS(t *testing.T) {
	t.Parallel()
	superRequestEditor := lib.SuperLogin(t)

	getBackendOASResp, err := lib.AdminClient.GetBackendOASWithResponse(
		t.Context(),
		"echo",
		adminapi.RequestEditorFn(superRequestEditor),
	)
	lib.CheckErr(err, t)
	lib.VerifyStatusCode(getBackendOASResp.StatusCode(), http.StatusOK, t)
}

// TestAdminGetFlowAsAdminUser verifies that a non-superuser admin user can also access the GetFlow endpoint.
func TestAdminGetFlowAsAdminUser(t *testing.T) {
	superRequestEditor := lib.SuperLogin(t)

	name := lib.Username()
	const pass = "password123"
	createResp, err := lib.AdminClient.CreateUserWithResponse(
		t.Context(),
		adminapi.CreateUserJSONRequestBody{Username: name, Password: pass},
		adminapi.RequestEditorFn(superRequestEditor),
	)
	lib.CheckErr(err, t)
	lib.VerifyStatusCode(createResp.StatusCode(), http.StatusCreated, t)

	userID := lib.MustGetAdminUserID(t, superRequestEditor, name)

	// Create a group with the flowviewer permission and assign the user to it.
	grpResp, err := lib.AdminClient.CreateGroupWithResponse(
		t.Context(),
		adminapi.CreateGroupJSONRequestBody{Name: lib.GroupName(), PermissionIDs: allPermissionIDs},
		adminapi.RequestEditorFn(superRequestEditor),
	)
	lib.CheckErr(err, t)
	lib.VerifyStatusCode(grpResp.StatusCode(), http.StatusCreated, t)

	updateResp, err := lib.AdminClient.UpdateUserGroupsWithResponse(
		t.Context(),
		userID,
		adminapi.UpdateUserGroupsJSONRequestBody{GroupIDs: []int{grpResp.JSON201.Id}},
		adminapi.RequestEditorFn(superRequestEditor),
	)
	lib.CheckErr(err, t)
	lib.VerifyStatusCode(updateResp.StatusCode(), http.StatusNoContent, t)

	adminRequestEditor := lib.AdminUserLogin(t, name, pass)

	getFlowResp, err := lib.AdminClient.GetFlowWithResponse(
		t.Context(),
		adminapi.RequestEditorFn(adminRequestEditor),
	)
	lib.CheckErr(err, t)
	lib.VerifyStatusCode(getFlowResp.StatusCode(), http.StatusOK, t)
}

// TestAdminGetBackendOASAsAdminUser verifies that a non-superuser admin user can also access the GetBackendOAS endpoint.
func TestAdminGetBackendOASAsAdminUser(t *testing.T) {
	superRequestEditor := lib.SuperLogin(t)

	name := lib.Username()
	const pass = "password123"
	createResp, err := lib.AdminClient.CreateUserWithResponse(
		t.Context(),
		adminapi.CreateUserJSONRequestBody{Username: name, Password: pass},
		adminapi.RequestEditorFn(superRequestEditor),
	)
	lib.CheckErr(err, t)
	lib.VerifyStatusCode(createResp.StatusCode(), http.StatusCreated, t)

	userID := lib.MustGetAdminUserID(t, superRequestEditor, name)

	// Create a group with the oasviewer permission and assign the user to it.
	grpResp, err := lib.AdminClient.CreateGroupWithResponse(
		t.Context(),
		adminapi.CreateGroupJSONRequestBody{Name: lib.GroupName(), PermissionIDs: allPermissionIDs},
		adminapi.RequestEditorFn(superRequestEditor),
	)
	lib.CheckErr(err, t)
	lib.VerifyStatusCode(grpResp.StatusCode(), http.StatusCreated, t)

	updateResp, err := lib.AdminClient.UpdateUserGroupsWithResponse(
		t.Context(),
		userID,
		adminapi.UpdateUserGroupsJSONRequestBody{GroupIDs: []int{grpResp.JSON201.Id}},
		adminapi.RequestEditorFn(superRequestEditor),
	)
	lib.CheckErr(err, t)
	lib.VerifyStatusCode(updateResp.StatusCode(), http.StatusNoContent, t)

	adminRequestEditor := lib.AdminUserLogin(t, name, pass)

	getBackendOASResp, err := lib.AdminClient.GetBackendOASWithResponse(
		t.Context(),
		"echo",
		adminapi.RequestEditorFn(adminRequestEditor),
	)
	lib.CheckErr(err, t)
	lib.VerifyStatusCode(getBackendOASResp.StatusCode(), http.StatusOK, t)
}
