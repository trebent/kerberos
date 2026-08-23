package security

const (
	// certDir is relative to the test working directory (test/suites/security/).
	certDir = "../../certs"

	adminUser         = "security-admin"
	adminUserPassword = "security-admin-password"

	basicAuthUser     = "security-basic-auth-user"
	basicAuthPassword = "security-basic-auth-password"
)

var orgID int64
