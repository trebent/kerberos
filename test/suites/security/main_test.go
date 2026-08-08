package security

import (
	"context"
	"crypto/tls"
	"fmt"
	"net/http"
	"os"
	"testing"
	"time"

	adminapi "github.com/trebent/kerberos/test/client/admin"
	authbasicapi "github.com/trebent/kerberos/test/client/auth/basic"
)

func TestMain(m *testing.M) {
	println("Running TestMain, setting up test foundation...")

	pool, err := getCAPool()
	if err != nil {
		panic(err)
	}

	httpClient := &http.Client{
		Timeout: 5 * time.Second,
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{
				RootCAs: pool,
			},
		},
	}

	adminClient, err := adminapi.NewClientWithResponses(
		fmt.Sprintf("https://%s:%d", getHost(), getAdminPort()),
		adminapi.WithHTTPClient(httpClient),
	)
	if err != nil {
		panic(err)
	}

	loginResp, err := adminClient.LoginSuperuserWithResponse(
		context.Background(), adminapi.LoginSuperuserJSONRequestBody{
			ClientId:     superUserClientID,
			ClientSecret: superUserClientSecret,
		},
	)
	if err != nil {
		panic(err)
	}
	if loginResp.StatusCode() != http.StatusNoContent {
		panic("superuser login response did not indicate success: " + loginResp.Status())
	}
	cookie, err := extractSessionCookie(loginResp.HTTPResponse)
	if err != nil {
		panic(err)
	}
	requestEditorSuper := makeRequestEditorFromCookie(cookie)

	createAdminUserResp, err := adminClient.CreateUserWithResponse(
		context.Background(),
		adminapi.CreateUserJSONRequestBody{
			Username: adminUser,
			Password: adminUserPassword,
		},
		adminapi.RequestEditorFn(requestEditorSuper),
	)
	if err != nil {
		panic(err)
	}
	if createAdminUserResp.StatusCode() != http.StatusCreated && createAdminUserResp.StatusCode() != http.StatusConflict {
		panic("create admin user response did not indicate success: " + createAdminUserResp.Status())
	}

	basicAuthClient, err := authbasicapi.NewClientWithResponses(
		fmt.Sprintf("https://%s:%d", getHost(), getAdminPort()),
		authbasicapi.WithHTTPClient(httpClient),
	)
	if err != nil {
		panic(err)
	}

	orgCreateResp, err := basicAuthClient.CreateOrganisationWithResponse(
		context.Background(),
		authbasicapi.CreateOrganisationJSONRequestBody{
			Name: "security-org",
		},
		authbasicapi.RequestEditorFn(requestEditorSuper),
	)
	if err != nil {
		panic(err)
	}
	if orgCreateResp.StatusCode() != http.StatusCreated && orgCreateResp.StatusCode() != http.StatusConflict {
		panic("create organization response did not indicate success: " + orgCreateResp.Status())
	}
	switch orgCreateResp.StatusCode() {
	case http.StatusCreated:
		orgID = orgCreateResp.JSON201.Id
	case http.StatusConflict:
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
			if org.Name == "security-org" {
				orgID = org.Id
				break
			}
		}
	default:
		panic("unexpected status code when creating organization: " + orgCreateResp.Status())
	}

	basicAuthResp, err := basicAuthClient.CreateUserWithResponse(
		context.Background(),
		orgID,
		authbasicapi.CreateUserJSONRequestBody{
			Name:     basicAuthUser,
			Password: basicAuthPassword,
		},
		authbasicapi.RequestEditorFn(requestEditorSuper),
	)
	if err != nil {
		panic(err)
	}
	if basicAuthResp.StatusCode() != http.StatusCreated && basicAuthResp.StatusCode() != http.StatusConflict {
		panic("create basic auth user response did not indicate success: " + basicAuthResp.Status())
	}

	println("Running tests...")
	code := m.Run()
	println("Testing done! Exit code:", code)
	os.Exit(code)
}
