package model

type (
	User struct {
		ID             int64
		Username       string
		Salt           string
		HashedPassword string
	}
	// SuperuserLoginUser holds the fields required to authenticate a superuser.
	SuperuserLoginUser struct {
		ID             int64
		Salt           string
		HashedPassword string
	}
	// UserAuth holds authentication credentials for password verification / update.
	UserAuth struct {
		Salt           string
		HashedPassword string
	}
	// GroupBinding represents a single user↔group association.
	GroupBinding struct {
		GroupID int64
		Name    string
	}
	Session struct {
		UserID    int64
		SessionID string
		IsSuper   bool
		Expires   int64

		// Permissions is populated for non-superusers to hold the permission IDs derived from their group memberships.
		// This is done at session validation time to avoid a database query on every request to fetch the user's permissions.
		Permissions []int64
	}
)
