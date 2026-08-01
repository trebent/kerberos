package security

import (
	"context"
	"crypto/tls"
	"fmt"
	"net/http"
	"os"
	"testing"
	"time"

	adminapi "github.com/trebent/kerberos/test/integration/client/admin"
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

	println("Running tests...")
	code := m.Run()
	println("Testing done! Exit code:", code)
	os.Exit(code)
}
