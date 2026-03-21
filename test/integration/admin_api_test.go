package integration

import (
	"net/http"
	"testing"

	adminapi "github.com/trebent/kerberos/ft/client/admin"
)

func TestAdminLoginSuperuser(t *testing.T) {
	t.Log("Logging the superuser in")
	superLoginResp, err := adminClient.LoginSuperuserWithResponse(
		t.Context(),
		adminapi.LoginSuperuserJSONRequestBody{ClientId: superUserClientID, ClientSecret: superUserClientSecret},
	)
	checkErr(err, t)
	verifyStatusCode(superLoginResp.StatusCode(), http.StatusNoContent, t)
	superSession := extractSession(superLoginResp.HTTPResponse, t)

	t.Log("Logging the superuser out")
	superLogoutResp, err := adminClient.LogoutSuperuserWithResponse(
		t.Context(),
		adminapi.RequestEditorFn(requestEditorSessionID(superSession)),
	)
	checkErr(err, t)
	verifyStatusCode(superLogoutResp.StatusCode(), http.StatusNoContent, t)
}

func TestAdminLoginSuperuserFailure(t *testing.T) {
	superLoginResp, err := adminClient.LoginSuperuserWithResponse(
		t.Context(),
		adminapi.LoginSuperuserJSONRequestBody{ClientId: "client-id", ClientSecret: "not-correct"},
	)
	checkErr(err, t)
	verifyStatusCode(superLoginResp.StatusCode(), http.StatusUnauthorized, t)

	// The generated client already parsed the response, so check the JSON401 response directly
	if superLoginResp.JSON401 != nil {
		if len(superLoginResp.JSON401.Errors) == 0 {
			t.Fatalf("Expected errors in response body, but got empty errors array")
		}
	} else {
		t.Fatalf("Expected JSON401 response but got nil")
	}
}

func TestAdminOASFailure(t *testing.T) {
	badSuperLoginResp, err := adminClient.LoginSuperuserWithResponse(t.Context(), adminapi.LoginSuperuserJSONRequestBody{})
	checkErr(err, t)
	verifyStatusCode(badSuperLoginResp.StatusCode(), http.StatusBadRequest, t)
	verifyAdminAPIErrorResponse(badSuperLoginResp.JSON400, t)
}

func TestAdminGetFlow(t *testing.T) {
	superLoginResp, err := adminClient.LoginSuperuserWithResponse(
		t.Context(),
		adminapi.LoginSuperuserJSONRequestBody{ClientId: superUserClientID, ClientSecret: superUserClientSecret},
	)
	checkErr(err, t)
	verifyStatusCode(superLoginResp.StatusCode(), http.StatusNoContent, t)
	superSession := extractSession(superLoginResp.HTTPResponse, t)

	getFlowResp, err := adminClient.GetFlowWithResponse(
		t.Context(),
		adminapi.RequestEditorFn(requestEditorSessionID(superSession)),
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
		case "router":
			if i != 1 {
				t.Error("router component should have index 1")
			}
			v := component.Data["backends"]
			dv := v.([]any)
			if len(dv) != 2 {
				t.Errorf("router component should have 2 backends, but got %v", len(dv))
			}
		case "custom":
			if i != 2 {
				t.Error("custom component should have index 2")
			}
			v := component.Data["component_count"]
			count := v.(float64)
			if count != 2 {
				t.Errorf("custom component should have component_count 2, but got %v", count)
			}
		case "authorizer":
			if i != 3 {
				t.Error("authorizer component should have index 3")
			}
			v := component.Data["order"]
			order := v.(float64)
			if order != 1 {
				t.Errorf("authorizer component should have order 1, but got %v", order)
			}
		case "oas-validator":
			if i != 4 {
				t.Error("oas-validator component should have index 4")
			}
			v := component.Data["order"]
			order := v.(float64)
			if order != 100 {
				t.Errorf("oas-validator component should have order 100, but got %v", order)
			}
		case "forwarder":
			if i != 5 {
				t.Error("forwarder component should have index 5")
			}
		default:
			t.Errorf("Unexpected flow component name: %s", component.Name)
		}
	}
}

func TestAdminGetBackendOAS(t *testing.T) {
	superLoginResp, err := adminClient.LoginSuperuserWithResponse(
		t.Context(),
		adminapi.LoginSuperuserJSONRequestBody{ClientId: superUserClientID, ClientSecret: superUserClientSecret},
	)
	checkErr(err, t)
	verifyStatusCode(superLoginResp.StatusCode(), http.StatusNoContent, t)
	superSession := extractSession(superLoginResp.HTTPResponse, t)

	getBackendOASResp, err := adminClient.GetBackendOASWithResponse(
		t.Context(),
		"echo",
		adminapi.RequestEditorFn(requestEditorSessionID(superSession)),
	)
	checkErr(err, t)
	verifyStatusCode(getBackendOASResp.StatusCode(), http.StatusOK, t)
}
