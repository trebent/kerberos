package integration

import (
	"net/http"
	"testing"

	adminapi "github.com/trebent/kerberos/ft/client/admin"
)

func TestAdminLoginSuperuser(t *testing.T) {
	superEditor := loginSuperuser(t)

	t.Log("Logging the superuser out")
	superLogoutResp, err := adminClient.LogoutSuperuserWithResponse(
		t.Context(),
		adminapi.RequestEditorFn(superEditor),
	)
	checkErr(err, t)
	verifyStatusCode(superLogoutResp.StatusCode(), http.StatusNoContent, t)

	t.Log("Running a GET flow request with the old session to verify it is invalidated")
	// Verify the old session is truly invalidated by attempting to access a protected endpoint with it.
	getFlowResp, err := adminClient.GetFlowWithResponse(
		t.Context(),
		adminapi.RequestEditorFn(requestEditorSessionID(superSession)),
	)
	checkErr(err, t)
	verifyStatusCode(getFlowResp.StatusCode(), http.StatusUnauthorized, t)
}

func TestAdminLoginSuperuserFailure(t *testing.T) {
	superLoginResp, err := adminClient.LoginSuperuserWithResponse(
		t.Context(),
		adminapi.LoginSuperuserJSONRequestBody{ClientId: superUserClientID, ClientSecret: "not-correct"},
	)
	checkErr(err, t)
	verifyStatusCode(superLoginResp.StatusCode(), http.StatusUnauthorized, t)
	verifyAdminAPIErrorResponse(superLoginResp.JSON401, t)
}

func TestAdminOASFailure(t *testing.T) {
	badSuperLoginResp, err := adminClient.LoginSuperuserWithResponse(
		t.Context(),
		adminapi.LoginSuperuserJSONRequestBody{},
	)
	checkErr(err, t)
	verifyStatusCode(badSuperLoginResp.StatusCode(), http.StatusBadRequest, t)
	verifyAdminAPIErrorResponse(badSuperLoginResp.JSON400, t)
}

func TestAdminGetFlow(t *testing.T) {
	superEditor := loginSuperuser(t)

	getFlowResp, err := adminClient.GetFlowWithResponse(
		t.Context(),
		adminapi.RequestEditorFn(superEditor),
	)
	checkErr(err, t)
	verifyStatusCode(getFlowResp.StatusCode(), http.StatusOK, t)

	for i, component := range *getFlowResp.JSON200 {
		t.Logf("Flow component index: %d name: %s", i, component.Name)

		switch component.Name {
		case "observability":
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
	superLoginResp, err := adminClient.LoginSuperuserWithResponse(
		t.Context(),
		adminapi.LoginSuperuserJSONRequestBody{ClientId: superUserClientID, ClientSecret: superUserClientSecret},
	)
	checkErr(err, t)
	verifyStatusCode(superLoginResp.StatusCode(), http.StatusNoContent, t)
	superSession := extractSession(superLoginResp.HTTPResponse, t)

	resp, err := adminClient.GetBackendOASWithResponse(
		t.Context(),
		"nonexistent-backend",
		adminapi.RequestEditorFn(requestEditorSessionID(superSession)),
	)
	checkErr(err, t)
	verifyStatusCode(resp.StatusCode(), http.StatusNotFound, t)
	verifyAdminAPIErrorResponse(resp.JSON404, t)
}

func TestAdminGetBackendOAS(t *testing.T) {
	superEditor := loginSuperuser(t)

	getBackendOASResp, err := adminClient.GetBackendOASWithResponse(
		t.Context(),
		"echo",
		adminapi.RequestEditorFn(superEditor),
	)
	checkErr(err, t)
	verifyStatusCode(getBackendOASResp.StatusCode(), http.StatusOK, t)
}

func TestAdminGetBackendOASNotFound(t *testing.T) {
	superEditor := loginSuperuser(t)

	getBackendOASResp, err := adminClient.GetBackendOASWithResponse(
		t.Context(),
		"non-existent-backend",
		adminapi.RequestEditorFn(superEditor),
	)
	checkErr(err, t)
	verifyStatusCode(getBackendOASResp.StatusCode(), http.StatusNotFound, t)
}
