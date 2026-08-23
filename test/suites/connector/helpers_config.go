package connector

import (
	"fmt"

	adminapi "github.com/trebent/kerberos/test/client/admin"
)

const (
	adminUser         = "connector-admin"
	adminUserPassword = "connector-admin-password"

	superUserClientID     = "admin"
	superUserClientSecret = "secret"
)

var adminClient, _ = adminapi.NewClientWithResponses(
	fmt.Sprintf("http://%s:%d", getHost(), getAdminPort()),
)
