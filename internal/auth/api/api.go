package api

type API interface {
	// User mgmt.
	SignUp()
	Login()
	Logout()

	// Group mgmt.
	CreateGroup()
	GetGroup()
	ListGroups()
	DeleteGroup()
	UpdateGroup()
	AddUserToGroup()
	RemoveUserFromGroup()

	// Organisation mgmt.
	CreateOrganisation()
	GetOrganisation()
	ListOrganisations()
	DeleteOrganisation()
	UpdateOrganisation()
}
