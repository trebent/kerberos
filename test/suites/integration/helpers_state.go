package integration

import (
	"fmt"
	"net/http"
	"sync/atomic"
	"time"

	adminapi "github.com/trebent/kerberos/test/client/admin"
	authbasicapi "github.com/trebent/kerberos/test/client/auth/basic"
)

var (
	client = &http.Client{Timeout: 4 * time.Second}

	basicAuthClient, _ = authbasicapi.NewClientWithResponses(
		fmt.Sprintf("http://%s:%d", getHost(), getAdminPort()),
	)
	adminClient, _ = adminapi.NewClientWithResponses(
		fmt.Sprintf("http://%s:%d", getHost(), getAdminPort()),
	)

	alwaysOrgID        = 0
	alwaysUserID       = 0
	alwaysGroupStaffID = 0
	alwaysGroupPlebID  = 0
	alwaysGroupDevID   = 0

	// Used to generate unique names.
	// This is initialised with a random int32 in TestMain.
	a = atomic.Int32{}
)

const (
	superUserClientID     = "admin"
	superUserClientSecret = "secret"
)

const (
	orgNameBase   = "Org"
	usernameBase  = "Smith"
	groupNameBase = "Group"

	// Always resource names, used to denote resource that all tests can expect to be present.
	// Always resource must never be altered or deleted by test cases, and are set up by test main.
	alwaysOrg          = "always"
	alwaysUser         = "always"
	alwaysAdminUser    = "always"
	alwaysUserPassword = "password123"
	alwaysGroupStaff   = "staff"
	alwaysGroupPleb    = "pleb"
	alwaysGroupDev     = "dev"
)

// Returns a guaranteed unique username.
func username() string {
	return fmt.Sprintf("%s-%d", usernameBase, a.Add(1))
}

// Returns a guaranteed unique org name.
func orgName() string {
	return fmt.Sprintf("%s-%d", orgNameBase, a.Add(1))
}

// Returns a guaranteed unique group name.
func groupName() string {
	return fmt.Sprintf("%s-%d", groupNameBase, a.Add(1))
}
