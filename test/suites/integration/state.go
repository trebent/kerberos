package integration

var (
	alwaysOrgID        = 0
	alwaysUserID       = 0
	alwaysGroupStaffID = 0
	alwaysGroupPlebID  = 0
	alwaysGroupDevID   = 0
)

// allPermissionIDs is the base set of all available admin group permissions.
// Tests that create admin groups should include these to avoid breaking permission-gated endpoints.
var allPermissionIDs = []int{1, 2, 3, 4, 5, 6, 7}

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
