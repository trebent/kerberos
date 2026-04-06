package basic

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"github.com/trebent/kerberos/internal/db"
	authbasicapi "github.com/trebent/kerberos/internal/oapi/auth/basic"
	apierror "github.com/trebent/kerberos/internal/oapi/error"
	"github.com/trebent/kerberos/internal/util/password"
	"github.com/trebent/zerologr"
)

type impl struct {
	db db.SQLClient
}

var (
	_ authbasicapi.StrictServerInterface = (*impl)(nil)

	GenErrInternal = authbasicapi.APIErrorResponse{Errors: []string{apierror.ErrInternal.Error()}}
	GenErrConflict = authbasicapi.APIErrorResponse{Errors: []string{apierror.ErrConflict.Error()}}
)

func makeGenAPIError(msg string) authbasicapi.APIErrorResponse {
	return authbasicapi.APIErrorResponse{Errors: []string{msg}}
}

func newSSI(db db.SQLClient) authbasicapi.StrictServerInterface {
	return &impl{db}
}

// Login implements [StrictServerInterface].
func (i *impl) Login(
	ctx context.Context,
	req authbasicapi.LoginRequestObject,
) (authbasicapi.LoginResponseObject, error) {
	user, err := i.dbLoginLookup(ctx, req.OrgID, req.Body.Username)
	if errors.Is(err, errNoUser) {
		return authbasicapi.Login401JSONResponse(makeGenAPIError("Login failed.")), nil
	}
	if err != nil {
		zerologr.Error(err, "Failed to look up user for login")
		return authbasicapi.Login500JSONResponse(GenErrInternal), nil
	}

	if !password.Match(user.Salt, user.HashedPassword, req.Body.Password) {
		zerologr.Info("User login failed due to password mismatch")
		return authbasicapi.Login401JSONResponse(makeGenAPIError("Login failed.")), nil
	}
	zerologr.V(10).Info("User has logged in successfully", "username", req.Body.Username)

	sessionID := uuid.NewString()
	if err := i.dbCreateSession(ctx, user.ID, user.OrganisationID, sessionID); err != nil {
		zerologr.Error(err, "Failed to create session for user")
		return authbasicapi.Login500JSONResponse(GenErrInternal), nil
	}

	return authbasicapi.Login204Response{
		Headers: authbasicapi.Login204ResponseHeaders{
			XKrbSession: sessionID,
		},
	}, nil
}

// Logout implements [StrictServerInterface].
func (i *impl) Logout(
	ctx context.Context,
	req authbasicapi.LogoutRequestObject,
) (authbasicapi.LogoutResponseObject, error) {
	userID := userFromContext(ctx)
	if err := i.dbDeleteUserSessions(ctx, req.OrgID, userID); err != nil {
		zerologr.Error(err, "Failed to delete user sessions")
		return authbasicapi.Logout500JSONResponse(GenErrInternal), nil
	}

	return authbasicapi.Logout204Response{}, nil
}

// ChangePassword implements [StrictServerInterface].
func (i *impl) ChangePassword(
	ctx context.Context,
	req authbasicapi.ChangePasswordRequestObject,
) (authbasicapi.ChangePasswordResponseObject, error) {
	u, err := i.dbGetUserAuth(ctx, req.OrgID, req.UserID)
	if errors.Is(err, errNoUser) {
		return authbasicapi.ChangePassword401JSONResponse(
			makeGenAPIError("Failed to change user password."),
		), nil
	}
	if err != nil {
		zerologr.Error(err, "Failed to get user auth")
		return authbasicapi.ChangePassword500JSONResponse(GenErrInternal), nil
	}

	if !password.Match(u.Salt, u.HashedPassword, req.Body.OldPassword) {
		zerologr.Info("Mismatched old password")
		return authbasicapi.ChangePassword401JSONResponse(
			makeGenAPIError("Failed to change user password."),
		), nil
	}

	_, newSalt, newHashedPassword := password.Make(req.Body.Password)
	if err := i.dbUpdateUserPassword(ctx, req.UserID, newSalt, newHashedPassword); err != nil {
		zerologr.Error(err, "Failed to update user password")
		return authbasicapi.ChangePassword500JSONResponse(GenErrInternal), nil
	}

	return authbasicapi.ChangePassword204Response{}, nil
}

// CreateGroup implements [StrictServerInterface].
func (i *impl) CreateGroup(
	ctx context.Context,
	req authbasicapi.CreateGroupRequestObject,
) (authbasicapi.CreateGroupResponseObject, error) {
	id, err := i.dbCreateGroup(ctx, req.OrgID, req.Body.Name)
	if err != nil {
		if errors.Is(err, db.ErrUnique) {
			return authbasicapi.CreateGroup409JSONResponse(GenErrConflict), nil
		}
		zerologr.Error(err, "Failed to create group")
		return authbasicapi.CreateGroup500JSONResponse(GenErrInternal), nil
	}

	return authbasicapi.CreateGroup201JSONResponse{
		Id:   id,
		Name: req.Body.Name,
	}, nil
}

// CreateOrganisation implements [StrictServerInterface].
func (i *impl) CreateOrganisation(
	ctx context.Context,
	req authbasicapi.CreateOrganisationRequestObject,
) (authbasicapi.CreateOrganisationResponseObject, error) {
	orgID, adminUserID, adminUsername, adminPassword, err := i.dbCreateOrganisation(
		ctx,
		req.Body.Name,
	)
	if err != nil {
		if errors.Is(err, db.ErrUnique) {
			return authbasicapi.CreateOrganisation409JSONResponse(GenErrConflict), nil
		}
		zerologr.Error(err, "Failed to create organisation")
		return authbasicapi.CreateOrganisation500JSONResponse(GenErrInternal), nil
	}

	return authbasicapi.CreateOrganisation201JSONResponse{
		Id:            orgID,
		Name:          req.Body.Name,
		AdminUserId:   adminUserID,
		AdminPassword: adminPassword,
		AdminUsername: adminUsername,
	}, nil
}

// CreateUser implements [StrictServerInterface].
func (i *impl) CreateUser(
	ctx context.Context,
	req authbasicapi.CreateUserRequestObject,
) (authbasicapi.CreateUserResponseObject, error) {
	_, salt, hashedPassword := password.Make(req.Body.Password)
	id, err := i.dbCreateUser(ctx, req.Body.Name, salt, hashedPassword, req.OrgID, false)
	if err != nil {
		if errors.Is(err, db.ErrUnique) {
			return authbasicapi.CreateUser409JSONResponse(GenErrConflict), nil
		}
		zerologr.Error(err, "Failed to create user")
		return authbasicapi.CreateUser500JSONResponse(GenErrInternal), nil
	}

	return authbasicapi.CreateUser201JSONResponse{
		Id:   id,
		Name: req.Body.Name,
	}, nil
}

// DeleteGroup implements [StrictServerInterface].
func (i *impl) DeleteGroup(
	ctx context.Context,
	req authbasicapi.DeleteGroupRequestObject,
) (authbasicapi.DeleteGroupResponseObject, error) {
	if err := i.dbDeleteGroup(ctx, req.OrgID, req.GroupID); err != nil {
		zerologr.Error(err, "Failed to delete group")
		return authbasicapi.DeleteGroup500JSONResponse(GenErrInternal), nil
	}

	return authbasicapi.DeleteGroup204Response{}, nil
}

// DeleteOrganisation implements [StrictServerInterface].
func (i *impl) DeleteOrganisation(
	ctx context.Context,
	req authbasicapi.DeleteOrganisationRequestObject,
) (authbasicapi.DeleteOrganisationResponseObject, error) {
	if err := i.dbDeleteOrg(ctx, req.OrgID); err != nil {
		zerologr.Error(err, "Failed to delete organisation")
		return authbasicapi.DeleteOrganisation500JSONResponse(GenErrInternal), nil
	}

	return authbasicapi.DeleteOrganisation204Response{}, nil
}

// DeleteUser implements [StrictServerInterface].
func (i *impl) DeleteUser(
	ctx context.Context,
	req authbasicapi.DeleteUserRequestObject,
) (authbasicapi.DeleteUserResponseObject, error) {
	if err := i.dbDeleteUser(ctx, req.OrgID, req.UserID); err != nil {
		zerologr.Error(err, "Failed to delete user")
		return authbasicapi.DeleteUser500JSONResponse(GenErrInternal), nil
	}

	return authbasicapi.DeleteUser204Response{}, nil
}

// GetGroup implements [StrictServerInterface].
func (i *impl) GetGroup(
	ctx context.Context,
	req authbasicapi.GetGroupRequestObject,
) (authbasicapi.GetGroupResponseObject, error) {
	g, err := i.dbGetGroup(ctx, req.OrgID, req.GroupID)
	if errors.Is(err, errNoGroup) {
		return authbasicapi.GetGroup404Response{}, nil
	}
	if err != nil {
		zerologr.Error(err, "Failed to get group")
		return authbasicapi.GetGroup500JSONResponse(GenErrInternal), nil
	}

	return authbasicapi.GetGroup200JSONResponse{Id: g.Id, Name: g.Name}, nil
}

// GetOrganisation implements [StrictServerInterface].
func (i *impl) GetOrganisation(
	ctx context.Context,
	req authbasicapi.GetOrganisationRequestObject,
) (authbasicapi.GetOrganisationResponseObject, error) {
	o, err := i.dbGetOrg(ctx, req.OrgID)
	if errors.Is(err, errNoOrg) {
		return authbasicapi.GetOrganisation404Response{}, nil
	}
	if err != nil {
		zerologr.Error(err, "Failed to get organisation")
		return authbasicapi.GetOrganisation500JSONResponse(GenErrInternal), nil
	}

	return authbasicapi.GetOrganisation200JSONResponse{Id: o.Id, Name: o.Name}, nil
}

// GetUser implements [StrictServerInterface].
func (i *impl) GetUser(
	ctx context.Context,
	req authbasicapi.GetUserRequestObject,
) (authbasicapi.GetUserResponseObject, error) {
	u, err := i.dbGetUser(ctx, req.OrgID, req.UserID)
	if errors.Is(err, errNoUser) {
		return authbasicapi.GetUser404Response{}, nil
	}
	if err != nil {
		zerologr.Error(err, "Failed to get user")
		return authbasicapi.GetUser500JSONResponse(GenErrInternal), nil
	}

	return authbasicapi.GetUser200JSONResponse{Id: u.Id, Name: u.Name}, nil
}

// GetUserGroups implements [StrictServerInterface].
func (i *impl) GetUserGroups(
	ctx context.Context,
	req authbasicapi.GetUserGroupsRequestObject,
) (authbasicapi.GetUserGroupsResponseObject, error) {
	groups, err := dbGetUserGroupNames(ctx, i.db, req.OrgID, req.UserID)
	if err != nil {
		zerologr.Error(err, "Failed to get user groups")
		return authbasicapi.GetUserGroups500JSONResponse(GenErrInternal), nil
	}

	return authbasicapi.GetUserGroups200JSONResponse(groups), nil
}

// ListGroups implements [StrictServerInterface].
func (i *impl) ListGroups(
	ctx context.Context,
	req authbasicapi.ListGroupsRequestObject,
) (authbasicapi.ListGroupsResponseObject, error) {
	groups, err := i.dbListGroups(ctx, req.OrgID)
	if err != nil {
		zerologr.Error(err, "Failed to list groups")
		return authbasicapi.ListGroups500JSONResponse(GenErrInternal), nil
	}

	return authbasicapi.ListGroups200JSONResponse(groups), nil
}

// ListOrganisations implements [StrictServerInterface].
func (i *impl) ListOrganisations(
	ctx context.Context,
	_ authbasicapi.ListOrganisationsRequestObject,
) (authbasicapi.ListOrganisationsResponseObject, error) {
	orgs, err := i.dbListOrgs(ctx)
	if err != nil {
		zerologr.Error(err, "Failed to list organisations")
		return authbasicapi.ListOrganisations500JSONResponse(GenErrInternal), nil
	}

	return authbasicapi.ListOrganisations200JSONResponse(orgs), nil
}

// ListUsers implements [StrictServerInterface].
func (i *impl) ListUsers(
	ctx context.Context,
	req authbasicapi.ListUsersRequestObject,
) (authbasicapi.ListUsersResponseObject, error) {
	users, err := i.dbListUsers(ctx, req.OrgID)
	if err != nil {
		zerologr.Error(err, "Failed to list users")
		return authbasicapi.ListUsers500JSONResponse(GenErrInternal), nil
	}

	return authbasicapi.ListUsers200JSONResponse(users), nil
}

// UpdateGroup implements [StrictServerInterface].
func (i *impl) UpdateGroup(
	ctx context.Context,
	req authbasicapi.UpdateGroupRequestObject,
) (authbasicapi.UpdateGroupResponseObject, error) {
	if _, err := i.dbGetGroup(ctx, req.OrgID, req.GroupID); errors.Is(err, errNoGroup) {
		return authbasicapi.UpdateGroup404Response{}, nil
	}

	if err := i.dbUpdateGroup(ctx, req.OrgID, req.GroupID, req.Body.Name); err != nil {
		if errors.Is(err, db.ErrUnique) {
			return authbasicapi.UpdateGroup409JSONResponse(GenErrConflict), nil
		}
		zerologr.Error(err, "Failed to update group")
		return authbasicapi.UpdateGroup500JSONResponse(GenErrInternal), nil
	}

	return authbasicapi.UpdateGroup200JSONResponse{Id: req.GroupID, Name: req.Body.Name}, nil
}

// UpdateOrganisation implements [StrictServerInterface].
func (i *impl) UpdateOrganisation(
	ctx context.Context,
	req authbasicapi.UpdateOrganisationRequestObject,
) (authbasicapi.UpdateOrganisationResponseObject, error) {
	if _, err := i.dbGetOrg(ctx, req.OrgID); errors.Is(err, errNoOrg) {
		return authbasicapi.UpdateOrganisation404Response{}, nil
	}

	if err := i.dbUpdateOrg(ctx, req.OrgID, req.Body.Name); err != nil {
		if errors.Is(err, db.ErrUnique) {
			return authbasicapi.UpdateOrganisation409JSONResponse(GenErrConflict), nil
		}
		zerologr.Error(err, "Failed to update organisation")
		return authbasicapi.UpdateOrganisation500JSONResponse(GenErrInternal), nil
	}

	return authbasicapi.UpdateOrganisation200JSONResponse{Id: req.OrgID, Name: req.Body.Name}, nil
}

// UpdateUser implements [StrictServerInterface].
func (i *impl) UpdateUser(
	ctx context.Context,
	req authbasicapi.UpdateUserRequestObject,
) (authbasicapi.UpdateUserResponseObject, error) {
	if _, err := i.dbGetUser(ctx, req.OrgID, req.UserID); errors.Is(err, errNoUser) {
		return authbasicapi.UpdateUser404Response{}, nil
	}

	if err := i.dbUpdateUser(ctx, req.OrgID, req.UserID, req.Body.Name); err != nil {
		if errors.Is(err, db.ErrUnique) {
			return authbasicapi.UpdateUser409JSONResponse(GenErrConflict), nil
		}
		zerologr.Error(err, "Failed to update user")
		return authbasicapi.UpdateUser500JSONResponse(GenErrInternal), nil
	}

	return authbasicapi.UpdateUser200JSONResponse{Id: req.UserID, Name: req.Body.Name}, nil
}

// UpdateUserGroups implements [StrictServerInterface].
func (i *impl) UpdateUserGroups(
	ctx context.Context,
	req authbasicapi.UpdateUserGroupsRequestObject,
) (authbasicapi.UpdateUserGroupsResponseObject, error) {
	if err := i.dbUpdateUserGroupBindings(ctx, req.OrgID, req.UserID, *req.Body); err != nil {
		zerologr.Error(err, "Failed to update user groups")
		return authbasicapi.UpdateUserGroups500JSONResponse(GenErrInternal), nil
	}

	return authbasicapi.UpdateUserGroups200JSONResponse(*req.Body), nil
}
