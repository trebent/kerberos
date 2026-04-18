package basic

import (
	"context"
	"errors"
	"net/http"

	"github.com/google/uuid"
	"github.com/trebent/kerberos/internal/db"
	authbasicapi "github.com/trebent/kerberos/internal/oapi/auth/basic"
	"github.com/trebent/kerberos/internal/util/password"
	"github.com/trebent/zerologr"
)

type impl struct {
	db db.SQLClient
}

var (
	_ authbasicapi.StrictServerInterface = (*impl)(nil)

	apiErrInternal     = makeGenAPIError(http.StatusText(http.StatusInternalServerError))
	apiErrConflict     = makeGenAPIError(http.StatusText(http.StatusConflict))
	apiErrUnauthorized = makeGenAPIError(http.StatusText(http.StatusUnauthorized))
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
	user, err := dbLoginLookup(ctx, i.db, req.OrgID, req.Body.Username)
	if errors.Is(err, errNoUser) {
		return authbasicapi.Login401JSONResponse(apiErrUnauthorized), nil
	}
	if err != nil {
		zerologr.Error(err, "Failed to look up user for login")
		return authbasicapi.Login500JSONResponse(apiErrInternal), nil
	}

	if !password.Match(user.Salt, user.HashedPassword, req.Body.Password) {
		zerologr.Info("User login failed due to password mismatch")
		return authbasicapi.Login401JSONResponse(apiErrUnauthorized), nil
	}
	zerologr.V(10).Info("User has logged in successfully", "username", req.Body.Username)

	sessionID := uuid.NewString()
	if err := dbCreateSession(ctx, i.db, user.ID, user.OrganisationID, sessionID); err != nil {
		zerologr.Error(err, "Failed to create session for user")
		return authbasicapi.Login500JSONResponse(apiErrInternal), nil
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
	if err := dbDeleteUserSessions(ctx, i.db, req.OrgID, userID); err != nil {
		zerologr.Error(err, "Failed to delete user sessions")
		return authbasicapi.Logout500JSONResponse(apiErrInternal), nil
	}

	return authbasicapi.Logout204Response{}, nil
}

// ChangePassword implements [StrictServerInterface].
func (i *impl) ChangePassword(
	ctx context.Context,
	req authbasicapi.ChangePasswordRequestObject,
) (authbasicapi.ChangePasswordResponseObject, error) {
	u, err := dbGetUserAuth(ctx, i.db, req.OrgID, req.UserID)
	if errors.Is(err, errNoUser) {
		return authbasicapi.ChangePassword401JSONResponse(apiErrUnauthorized), nil
	}
	if err != nil {
		zerologr.Error(err, "Failed to get user auth")
		return authbasicapi.ChangePassword500JSONResponse(apiErrInternal), nil
	}

	if !password.Match(u.Salt, u.HashedPassword, req.Body.OldPassword) {
		zerologr.Info("Mismatched old password")
		return authbasicapi.ChangePassword401JSONResponse(apiErrUnauthorized), nil
	}

	_, newSalt, newHashedPassword := password.Make(req.Body.Password)
	if err := dbUpdateUserPassword(ctx, i.db, req.UserID, newSalt, newHashedPassword); err != nil {
		zerologr.Error(err, "Failed to update user password")
		return authbasicapi.ChangePassword500JSONResponse(apiErrInternal), nil
	}

	return authbasicapi.ChangePassword204Response{}, nil
}

// CreateGroup implements [StrictServerInterface].
func (i *impl) CreateGroup(
	ctx context.Context,
	req authbasicapi.CreateGroupRequestObject,
) (authbasicapi.CreateGroupResponseObject, error) {
	id, err := dbCreateGroup(ctx, i.db, req.OrgID, req.Body.Name)
	if err != nil {
		if errors.Is(err, db.ErrUnique) {
			return authbasicapi.CreateGroup409JSONResponse(apiErrConflict), nil
		}
		zerologr.Error(err, "Failed to create group")
		return authbasicapi.CreateGroup500JSONResponse(apiErrInternal), nil
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
	orgID, adminUserID, adminUsername, adminPassword, err := dbCreateOrganisation(
		ctx,
		i.db,
		req.Body.Name,
	)
	if err != nil {
		if errors.Is(err, db.ErrUnique) {
			return authbasicapi.CreateOrganisation409JSONResponse(apiErrConflict), nil
		}
		zerologr.Error(err, "Failed to create organisation")
		return authbasicapi.CreateOrganisation500JSONResponse(apiErrInternal), nil
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
	id, err := dbCreateUser(ctx, i.db, req.Body.Name, salt, hashedPassword, req.OrgID)
	if err != nil {
		if errors.Is(err, db.ErrUnique) {
			return authbasicapi.CreateUser409JSONResponse(apiErrConflict), nil
		}
		zerologr.Error(err, "Failed to create user")
		return authbasicapi.CreateUser500JSONResponse(apiErrInternal), nil
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
	if err := dbDeleteGroup(ctx, i.db, req.OrgID, req.GroupID); err != nil {
		zerologr.Error(err, "Failed to delete group")
		return authbasicapi.DeleteGroup500JSONResponse(apiErrInternal), nil
	}

	return authbasicapi.DeleteGroup204Response{}, nil
}

// DeleteOrganisation implements [StrictServerInterface].
func (i *impl) DeleteOrganisation(
	ctx context.Context,
	req authbasicapi.DeleteOrganisationRequestObject,
) (authbasicapi.DeleteOrganisationResponseObject, error) {
	if err := dbDeleteOrg(ctx, i.db, req.OrgID); err != nil {
		zerologr.Error(err, "Failed to delete organisation")
		return authbasicapi.DeleteOrganisation500JSONResponse(apiErrInternal), nil
	}

	return authbasicapi.DeleteOrganisation204Response{}, nil
}

// DeleteUser implements [StrictServerInterface].
func (i *impl) DeleteUser(
	ctx context.Context,
	req authbasicapi.DeleteUserRequestObject,
) (authbasicapi.DeleteUserResponseObject, error) {
	if err := dbDeleteUser(ctx, i.db, req.OrgID, req.UserID); err != nil {
		zerologr.Error(err, "Failed to delete user")
		return authbasicapi.DeleteUser500JSONResponse(apiErrInternal), nil
	}

	return authbasicapi.DeleteUser204Response{}, nil
}

// GetGroup implements [StrictServerInterface].
func (i *impl) GetGroup(
	ctx context.Context,
	req authbasicapi.GetGroupRequestObject,
) (authbasicapi.GetGroupResponseObject, error) {
	g, err := dbGetGroup(ctx, i.db, req.OrgID, req.GroupID)
	if errors.Is(err, errNoGroup) {
		return authbasicapi.GetGroup404Response{}, nil
	}
	if err != nil {
		zerologr.Error(err, "Failed to get group")
		return authbasicapi.GetGroup500JSONResponse(apiErrInternal), nil
	}

	return authbasicapi.GetGroup200JSONResponse{Id: g.Id, Name: g.Name}, nil
}

// GetOrganisation implements [StrictServerInterface].
func (i *impl) GetOrganisation(
	ctx context.Context,
	req authbasicapi.GetOrganisationRequestObject,
) (authbasicapi.GetOrganisationResponseObject, error) {
	o, err := dbGetOrg(ctx, i.db, req.OrgID)
	if errors.Is(err, errNoOrg) {
		return authbasicapi.GetOrganisation404Response{}, nil
	}
	if err != nil {
		zerologr.Error(err, "Failed to get organisation")
		return authbasicapi.GetOrganisation500JSONResponse(apiErrInternal), nil
	}

	return authbasicapi.GetOrganisation200JSONResponse{Id: o.Id, Name: o.Name}, nil
}

// GetUser implements [StrictServerInterface].
func (i *impl) GetUser(
	ctx context.Context,
	req authbasicapi.GetUserRequestObject,
) (authbasicapi.GetUserResponseObject, error) {
	u, err := dbGetUser(ctx, i.db, req.OrgID, req.UserID)
	if errors.Is(err, errNoUser) {
		return authbasicapi.GetUser404Response{}, nil
	}
	if err != nil {
		zerologr.Error(err, "Failed to get user")
		return authbasicapi.GetUser500JSONResponse(apiErrInternal), nil
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
		return authbasicapi.GetUserGroups500JSONResponse(apiErrInternal), nil
	}

	return authbasicapi.GetUserGroups200JSONResponse(groups), nil
}

// ListGroups implements [StrictServerInterface].
func (i *impl) ListGroups(
	ctx context.Context,
	req authbasicapi.ListGroupsRequestObject,
) (authbasicapi.ListGroupsResponseObject, error) {
	groups, err := dbListGroups(ctx, i.db, req.OrgID)
	if err != nil {
		zerologr.Error(err, "Failed to list groups")
		return authbasicapi.ListGroups500JSONResponse(apiErrInternal), nil
	}

	return authbasicapi.ListGroups200JSONResponse(groups), nil
}

// ListOrganisations implements [StrictServerInterface].
func (i *impl) ListOrganisations(
	ctx context.Context,
	_ authbasicapi.ListOrganisationsRequestObject,
) (authbasicapi.ListOrganisationsResponseObject, error) {
	orgs, err := dbListOrgs(ctx, i.db)
	if err != nil {
		zerologr.Error(err, "Failed to list organisations")
		return authbasicapi.ListOrganisations500JSONResponse(apiErrInternal), nil
	}

	return authbasicapi.ListOrganisations200JSONResponse(orgs), nil
}

// ListUsers implements [StrictServerInterface].
func (i *impl) ListUsers(
	ctx context.Context,
	req authbasicapi.ListUsersRequestObject,
) (authbasicapi.ListUsersResponseObject, error) {
	users, err := dbListUsers(ctx, i.db, req.OrgID)
	if err != nil {
		zerologr.Error(err, "Failed to list users")
		return authbasicapi.ListUsers500JSONResponse(apiErrInternal), nil
	}

	return authbasicapi.ListUsers200JSONResponse(users), nil
}

// UpdateGroup implements [StrictServerInterface].
func (i *impl) UpdateGroup(
	ctx context.Context,
	req authbasicapi.UpdateGroupRequestObject,
) (authbasicapi.UpdateGroupResponseObject, error) {
	if _, err := dbGetGroup(ctx, i.db, req.OrgID, req.GroupID); errors.Is(err, errNoGroup) {
		return authbasicapi.UpdateGroup404Response{}, nil
	}

	if err := dbUpdateGroup(ctx, i.db, req.OrgID, req.GroupID, req.Body.Name); err != nil {
		if errors.Is(err, db.ErrUnique) {
			return authbasicapi.UpdateGroup409JSONResponse(apiErrConflict), nil
		}
		zerologr.Error(err, "Failed to update group")
		return authbasicapi.UpdateGroup500JSONResponse(apiErrInternal), nil
	}

	return authbasicapi.UpdateGroup200JSONResponse{Id: req.GroupID, Name: req.Body.Name}, nil
}

// UpdateOrganisation implements [StrictServerInterface].
func (i *impl) UpdateOrganisation(
	ctx context.Context,
	req authbasicapi.UpdateOrganisationRequestObject,
) (authbasicapi.UpdateOrganisationResponseObject, error) {
	if _, err := dbGetOrg(ctx, i.db, req.OrgID); errors.Is(err, errNoOrg) {
		return authbasicapi.UpdateOrganisation404Response{}, nil
	}

	if err := dbUpdateOrg(ctx, i.db, req.OrgID, req.Body.Name); err != nil {
		if errors.Is(err, db.ErrUnique) {
			return authbasicapi.UpdateOrganisation409JSONResponse(apiErrConflict), nil
		}
		zerologr.Error(err, "Failed to update organisation")
		return authbasicapi.UpdateOrganisation500JSONResponse(apiErrInternal), nil
	}

	return authbasicapi.UpdateOrganisation200JSONResponse{Id: req.OrgID, Name: req.Body.Name}, nil
}

// UpdateUser implements [StrictServerInterface].
func (i *impl) UpdateUser(
	ctx context.Context,
	req authbasicapi.UpdateUserRequestObject,
) (authbasicapi.UpdateUserResponseObject, error) {
	if _, err := dbGetUser(ctx, i.db, req.OrgID, req.UserID); errors.Is(err, errNoUser) {
		return authbasicapi.UpdateUser404Response{}, nil
	}

	if err := dbUpdateUser(ctx, i.db, req.OrgID, req.UserID, req.Body.Name); err != nil {
		if errors.Is(err, db.ErrUnique) {
			return authbasicapi.UpdateUser409JSONResponse(apiErrConflict), nil
		}
		zerologr.Error(err, "Failed to update user")
		return authbasicapi.UpdateUser500JSONResponse(apiErrInternal), nil
	}

	return authbasicapi.UpdateUser200JSONResponse{Id: req.UserID, Name: req.Body.Name}, nil
}

// UpdateUserGroups implements [StrictServerInterface].
func (i *impl) UpdateUserGroups(
	ctx context.Context,
	req authbasicapi.UpdateUserGroupsRequestObject,
) (authbasicapi.UpdateUserGroupsResponseObject, error) {
	if err := dbUpdateUserGroupBindings(ctx, i.db, req.OrgID, req.UserID, *req.Body); err != nil {
		zerologr.Error(err, "Failed to update user groups")
		return authbasicapi.UpdateUserGroups500JSONResponse(apiErrInternal), nil
	}

	return authbasicapi.UpdateUserGroups200JSONResponse(*req.Body), nil
}
