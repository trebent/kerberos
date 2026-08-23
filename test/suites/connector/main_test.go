package connector

import (
	"context"
	lib "github.com/trebent/kerberos/test/lib"
	"net/http"
	"os"
	"testing"

	adminapi "github.com/trebent/kerberos/test/client/admin"
)

func TestMain(m *testing.M) {
	println("Running TestMain, setting up test foundation...")

	loginResp, err := lib.AdminClient.LoginSuperuserWithResponse(
		context.Background(), adminapi.LoginSuperuserJSONRequestBody{
			ClientId:     lib.SuperUserClientID,
			ClientSecret: lib.SuperUserClientSecret,
		},
	)
	if err != nil {
		panic(err)
	}
	if loginResp.StatusCode() != http.StatusNoContent {
		panic("superuser login response did not indicate success: " + loginResp.Status())
	}
	cookie, err := lib.ExtractSessionCookie(loginResp.HTTPResponse)
	if err != nil {
		panic(err)
	}
	requestEditorSuper := lib.MakeRequestEditorFromCookie(cookie)

	createAdminUserResp, err := lib.AdminClient.CreateUserWithResponse(
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
