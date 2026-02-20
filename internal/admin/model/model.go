package model

type (
	User struct {
		ID             int64
		Name           string
		Salt           string
		HashedPassword string
		Administrator  bool
		SuperUser      bool
	}
)
