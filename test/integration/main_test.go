package integration

import (
	"context"
	"net/http"
	"os"
	"testing"

	authadminapi "github.com/trebent/kerberos/ft/client/auth/admin"
	authbasicapi "github.com/trebent/kerberos/ft/client/auth/basic"
)

func TestMain(m *testing.M) {
	loginResp, err := adminClient.LoginSuperuserWithResponse(
		context.Background(), authadminapi.LoginSuperuserJSONRequestBody{ClientId: "client-id", ClientSecret: "client-secret"},
	)
	if err != nil {
		panic(err)
	}
	if loginResp.StatusCode() != http.StatusNoContent {
		panic("superuser login response did not indicate success: " + loginResp.Status())
	}
	requestEditorSuper := requestEditorSessionID(loginResp.HTTPResponse.Header.Get("x-krb-session"))

	orgResp, err := basicAuthClient.CreateOrganisationWithResponse(
		context.Background(),
		authbasicapi.CreateOrganisationRequest{Name: alwaysOrg},
		authbasicapi.RequestEditorFn(requestEditorSuper),
	)
	if err != nil {
		panic(err)
	}
	if orgResp.StatusCode() != http.StatusCreated {
		if orgResp.StatusCode() == http.StatusConflict {
			orgListResp, err := basicAuthClient.ListOrganisationsWithResponse(
				context.Background(),
				authbasicapi.RequestEditorFn(requestEditorSuper),
			)
			if err != nil {
				panic(err)
			}
			if orgListResp.StatusCode() != http.StatusOK {
				panic("org list response was not OK: " + orgListResp.Status())
			}
			for _, org := range *orgListResp.JSON200 {
				if org.Name == alwaysOrg {
					alwaysOrgID = int(org.Id)
					break
				}
			}
			if alwaysOrgID == 0 {
				panic("did not find always org")
			}
		} else {
			panic("org create response was neither created nor conflict: " + orgResp.Status())
		}
	} else {
		alwaysOrgID = int(orgResp.JSON201.Id)
	}

	userResp, err := basicAuthClient.CreateUserWithResponse(
		context.Background(),
		authbasicapi.Orgid(alwaysOrgID),
		authbasicapi.CreateUserRequest{Name: alwaysUser, Password: "1234567890"},
		authbasicapi.RequestEditorFn(requestEditorSuper),
	)
	if err != nil {
		panic(err)
	}
	if userResp.StatusCode() != http.StatusCreated {
		if userResp.StatusCode() == http.StatusConflict {
			userListResp, err := basicAuthClient.ListUsersWithResponse(
				context.Background(),
				authbasicapi.Orgid(alwaysOrgID),
				authbasicapi.RequestEditorFn(requestEditorSuper),
			)
			if err != nil {
				panic(err)
			}
			if userListResp.StatusCode() != http.StatusOK {
				panic("user list response was not OK: " + userListResp.Status())
			}
			for _, user := range *userListResp.JSON200 {
				if user.Name == alwaysUser {
					alwaysUserID = int(user.Id)
					break
				}
			}
			if alwaysUserID == 0 {
				panic("did not find always user")
			}
		} else {
			panic("user create response was neither created nor conflict: " + userResp.Status())
		}
	} else {
		alwaysUserID = int(userResp.JSON201.Id)
	}

	alwaysGroupDevID = createOrGetGroup(alwaysGroupDev, authbasicapi.RequestEditorFn(requestEditorSuper))
	if alwaysGroupDevID == 0 {
		panic("failed to find dev group id")
	}
	alwaysGroupPlebID = createOrGetGroup(alwaysGroupPleb, authbasicapi.RequestEditorFn(requestEditorSuper))
	if alwaysGroupPlebID == 0 {
		panic("failed to find pleb group id")
	}
	alwaysGroupStaffID = createOrGetGroup(alwaysGroupStaff, authbasicapi.RequestEditorFn(requestEditorSuper))
	if alwaysGroupStaffID == 0 {
		panic("failed to find staff group id")
	}

	code := m.Run()
	os.Exit(code)
}

func createOrGetGroup(name string, requestEditor authbasicapi.RequestEditorFn) int {
	groupCreateResp, err := basicAuthClient.CreateGroupWithResponse(
		context.Background(),
		authbasicapi.Orgid(alwaysOrgID),
		authbasicapi.CreateGroupRequest{Name: name},
		requestEditor,
	)
	if err != nil {
		panic(err)
	}
	if groupCreateResp.StatusCode() != http.StatusCreated {
		if groupCreateResp.StatusCode() == http.StatusConflict {
			groupListResp, err := basicAuthClient.ListGroupsWithResponse(
				context.Background(),
				authbasicapi.Orgid(alwaysOrgID),
				requestEditor,
			)
			if err != nil {
				panic(err)
			}
			if groupListResp.StatusCode() != http.StatusOK {
				panic("group list response was not OK: " + groupListResp.Status())
			}

			for _, group := range *groupListResp.JSON200 {
				if group.Name == name {
					return int(group.Id)
				}
			}

			return 0
		}
	}

	return int(groupCreateResp.JSON201.Id)
}
