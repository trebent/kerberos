package model

type (
	User struct {
		ID             int64
		Username       string
		Salt           string
		HashedPassword string
	}
	Session struct {
		UserID    int64
		SessionID string
		IsSuper   bool
		Expires   int64
	}
)
