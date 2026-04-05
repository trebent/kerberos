package models

type (
	// Session holds the fields scanned from a session query.
	Session struct {
		UserID        int64
		OrgID         int64
		Administrator bool
		Expires       int64
	}

	// LoginUser holds the fields returned by selectLoginUser that are actually used.
	LoginUser struct {
		ID             int64
		Salt           string
		HashedPassword string
		OrganisationID int64
	}

	// UserAuth holds the password-related fields returned by selectFullUser.
	UserAuth struct {
		Salt           string
		HashedPassword string
	}

	// GroupBinding holds the ID and name of a group binding.
	GroupBinding struct {
		GroupID int64
		Name    string
	}

	Organisation struct {
		ID   int64
		Name string
	}
	User struct {
		ID             int64
		OrganisationID int64
		Name           string
		Salt           string
		HashedPassword string
		Administrator  bool
	}
)
