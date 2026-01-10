package model

type (
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
		SuperUser      bool
	}
)
