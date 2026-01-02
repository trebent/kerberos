package basic

import (
	"context"

	"github.com/trebent/kerberos/internal/db"
)

//go:generate go tool oapi-codegen -config oas/config.yaml -o ./gen.go oas/api.yaml

type impl struct {
	db db.SQLClient
}

var _ StrictServerInterface = (*impl)(nil)

func NewSSI(db db.SQLClient) StrictServerInterface {
	return &impl{db}
}

// Login implements [StrictServerInterface].
func (i *impl) Login(_ context.Context, _ LoginRequestObject) (LoginResponseObject, error) {
	panic("unimplemented")
}

// Logout implements [StrictServerInterface].
func (i *impl) Logout(
	_ context.Context,
	_ LogoutRequestObject,
) (LogoutResponseObject, error) {
	panic("unimplemented")
}

// ChangePassword implements [StrictServerInterface].
func (i *impl) ChangePassword(
	_ context.Context,
	_ ChangePasswordRequestObject,
) (ChangePasswordResponseObject, error) {
	panic("unimplemented")
}

// CreateGroup implements [StrictServerInterface].
func (i *impl) CreateGroup(
	_ context.Context,
	_ CreateGroupRequestObject,
) (CreateGroupResponseObject, error) {
	panic("unimplemented")
}

// CreateOrganisation implements [StrictServerInterface].
func (i *impl) CreateOrganisation(
	_ context.Context,
	_ CreateOrganisationRequestObject,
) (CreateOrganisationResponseObject, error) {
	panic("unimplemented")
}

// CreateUser implements [StrictServerInterface].
func (i *impl) CreateUser(
	_ context.Context,
	_ CreateUserRequestObject,
) (CreateUserResponseObject, error) {
	panic("unimplemented")
}

// DeleteGroup implements [StrictServerInterface].
func (i *impl) DeleteGroup(
	_ context.Context,
	_ DeleteGroupRequestObject,
) (DeleteGroupResponseObject, error) {
	panic("unimplemented")
}

// DeleteOrganisation implements [StrictServerInterface].
func (i *impl) DeleteOrganisation(
	_ context.Context,
	_ DeleteOrganisationRequestObject,
) (DeleteOrganisationResponseObject, error) {
	panic("unimplemented")
}

// DeleteUser implements [StrictServerInterface].
func (i *impl) DeleteUser(
	_ context.Context,
	_ DeleteUserRequestObject,
) (DeleteUserResponseObject, error) {
	panic("unimplemented")
}

// GetGroup implements [StrictServerInterface].
func (i *impl) GetGroup(
	_ context.Context,
	_ GetGroupRequestObject,
) (GetGroupResponseObject, error) {
	panic("unimplemented")
}

// GetOrganisation implements [StrictServerInterface].
func (i *impl) GetOrganisation(
	_ context.Context,
	_ GetOrganisationRequestObject,
) (GetOrganisationResponseObject, error) {
	panic("unimplemented")
}

// GetUser implements [StrictServerInterface].
func (i *impl) GetUser(
	_ context.Context,
	_ GetUserRequestObject,
) (GetUserResponseObject, error) {
	panic("unimplemented")
}

// GetUserGroups implements [StrictServerInterface].
func (i *impl) GetUserGroups(
	_ context.Context,
	_ GetUserGroupsRequestObject,
) (GetUserGroupsResponseObject, error) {
	panic("unimplemented")
}

// ListGroups implements [StrictServerInterface].
func (i *impl) ListGroups(
	_ context.Context,
	_ ListGroupsRequestObject,
) (ListGroupsResponseObject, error) {
	panic("unimplemented")
}

// ListOrganisations implements [StrictServerInterface].
func (i *impl) ListOrganisations(
	_ context.Context,
	_ ListOrganisationsRequestObject,
) (ListOrganisationsResponseObject, error) {
	panic("unimplemented")
}

// ListUsers implements [StrictServerInterface].
func (i *impl) ListUsers(
	_ context.Context,
	_ ListUsersRequestObject,
) (ListUsersResponseObject, error) {
	panic("unimplemented")
}

// UpdateGroup implements [StrictServerInterface].
func (i *impl) UpdateGroup(
	_ context.Context,
	_ UpdateGroupRequestObject,
) (UpdateGroupResponseObject, error) {
	panic("unimplemented")
}

// UpdateOrganisation implements [StrictServerInterface].
func (i *impl) UpdateOrganisation(
	_ context.Context,
	_ UpdateOrganisationRequestObject,
) (UpdateOrganisationResponseObject, error) {
	panic("unimplemented")
}

// UpdateUser implements [StrictServerInterface].
func (i *impl) UpdateUser(
	_ context.Context,
	_ UpdateUserRequestObject,
) (UpdateUserResponseObject, error) {
	panic("unimplemented")
}

// UpdateUserGroups implements [StrictServerInterface].
func (i *impl) UpdateUserGroups(
	_ context.Context,
	_ UpdateUserGroupsRequestObject,
) (UpdateUserGroupsResponseObject, error) {
	panic("unimplemented")
}
