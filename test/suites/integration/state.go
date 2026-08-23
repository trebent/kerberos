package integration

var (
	alwaysOrgID        = 0
	alwaysUserID       = 0
	alwaysGroupStaffID = 0
	alwaysGroupPlebID  = 0
	alwaysGroupDevID   = 0
)

const (
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
