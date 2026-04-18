package admin

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/trebent/kerberos/internal/admin/model"
	"github.com/trebent/kerberos/internal/db"
	adminapi "github.com/trebent/kerberos/internal/oapi/admin"
	apierror "github.com/trebent/kerberos/internal/oapi/error"
	"github.com/trebent/kerberos/internal/util/password"
	"github.com/trebent/zerologr"
)

// LoginSuperuser implements [StrictServerInterface].
func (i *impl) LoginSuperuser(
	ctx context.Context,
	request adminapi.LoginSuperuserRequestObject,
) (adminapi.LoginSuperuserResponseObject, error) {
	superuser, err := dbGetSuperuser(ctx, i.sqlClient)
	if err != nil {
		zerologr.Error(err, "Failed to query superuser")
		return adminapi.LoginSuperuser500JSONResponse(
			makeGenAPIError(apierror.APIErrInternal.Error()),
		), nil
	}

	if !password.Match(
		superuser.Salt,
		superuser.HashedPassword,
		request.Body.ClientSecret,
	) {
		return adminapi.LoginSuperuser401JSONResponse(makeGenAPIError("Login failed.")), nil
	}

	if superuser.Username != request.Body.ClientId {
		return adminapi.LoginSuperuser401JSONResponse(makeGenAPIError("Login failed.")), nil
	}

	sessionID := uuid.NewString()
	_, err = i.sqlClient.Exec(
		ctx,
		insertSession,
		sql.NamedArg{Name: "user_id", Value: superuser.ID},
		sql.NamedArg{Name: "session_id", Value: sessionID},
		sql.NamedArg{Name: "expires", Value: time.Now().Add(superSessionExpiry).UnixMilli()},
	)
	if err != nil {
		zerologr.Error(err, "Failed to store super-session")
		return adminapi.LoginSuperuser500JSONResponse(
			makeGenAPIError(apierror.APIErrInternal.Error()),
		), nil
	}

	return adminapi.LoginSuperuser204Response{
		Headers: adminapi.LoginSuperuser204ResponseHeaders{
			XKrbSession: sessionID,
		},
	}, nil
}

// LogoutSuperuser implements [StrictServerInterface].
func (i *impl) LogoutSuperuser(
	ctx context.Context,
	_ adminapi.LogoutSuperuserRequestObject,
) (adminapi.LogoutSuperuserResponseObject, error) {
	_, err := i.sqlClient.Exec(ctx, deleteSuperSessions)
	if err != nil {
		zerologr.Error(err, "Failed to delete super sessions during logout")
		return adminapi.LogoutSuperuser500JSONResponse(
			makeGenAPIError(apierror.APIErrInternal.Error()),
		), nil
	}
	return adminapi.LogoutSuperuser204Response{}, nil
}

// Login implements [withExtensions].
func (i *impl) Login(
	ctx context.Context,
	request adminapi.LoginRequestObject,
) (adminapi.LoginResponseObject, error) {
	u, err := dbLoginLookup(ctx, i.sqlClient, request.Body.Username)
	if err != nil {
		if errors.Is(err, errNoUser) {
			return adminapi.Login401JSONResponse{
				UnauthorizedErrorJSONResponse: makeErrUnauthorized("Login failed."),
			}, nil
		}
		zerologr.Error(err, "Failed to look up admin user during login")
		return adminapi.Login500JSONResponse{InternalErrorJSONResponse: apiErrInternal}, nil
	}

	if !password.Match(u.Salt, u.HashedPassword, request.Body.Password) {
		return adminapi.Login401JSONResponse{
			UnauthorizedErrorJSONResponse: makeErrUnauthorized("Login failed."),
		}, nil
	}

	sessionID := uuid.NewString()
	if err := dbCreateSession(ctx, i.sqlClient, u.ID, sessionID); err != nil {
		zerologr.Error(err, "Failed to store admin session")
		return adminapi.Login500JSONResponse{InternalErrorJSONResponse: apiErrInternal}, nil
	}

	return adminapi.Login204Response{
		Headers: adminapi.Login204ResponseHeaders{
			XKrbSession: sessionID,
		},
	}, nil
}

// Logout implements [withExtensions].
func (i *impl) Logout(
	ctx context.Context,
	_ adminapi.LogoutRequestObject,
) (adminapi.LogoutResponseObject, error) {
	session, ok := ctx.Value(adminContextSession).(*model.Session)
	if !ok || session == nil {
		return adminapi.Logout401JSONResponse{
			UnauthorizedErrorJSONResponse: makeErrUnauthorized(apierror.ErrNoSession.Error()),
		}, nil
	}

	if err := dbDeleteSession(ctx, i.sqlClient, session.SessionID); err != nil {
		zerologr.Error(err, "Failed to delete admin session during logout")
		return adminapi.Logout500JSONResponse{InternalErrorJSONResponse: apiErrInternal}, nil
	}

	return adminapi.Logout204Response{}, nil
}

// CreateUser implements [withExtensions].
func (i *impl) CreateUser(
	ctx context.Context,
	request adminapi.CreateUserRequestObject,
) (adminapi.CreateUserResponseObject, error) {
	_, salt, hashedPassword := password.Make(request.Body.Password)

	if _, err := dbCreateUser(
		ctx,
		i.sqlClient,
		request.Body.Username,
		salt,
		hashedPassword,
	); err != nil {
		if errors.Is(err, db.ErrUnique) {
			zerologr.Error(err, "Username conflict")
			return adminapi.CreateUser409JSONResponse{
				ConflictErrorJSONResponse: apiErrConflict,
			}, nil
		}

		zerologr.Error(err, "Failed to create admin user")
		return adminapi.CreateUser500JSONResponse{InternalErrorJSONResponse: apiErrInternal}, nil
	}

	return adminapi.CreateUser201Response{}, nil
}

// GetUsers implements [withExtensions].
func (i *impl) GetUsers(
	ctx context.Context,
	_ adminapi.GetUsersRequestObject,
) (adminapi.GetUsersResponseObject, error) {
	users, err := dbListUsers(ctx, i.sqlClient)
	if err != nil {
		zerologr.Error(err, "Failed to list admin users")
		return adminapi.GetUsers500JSONResponse{InternalErrorJSONResponse: apiErrInternal}, nil
	}

	return adminapi.GetUsers200JSONResponse(users), nil
}

// GetUser implements [withExtensions].
func (i *impl) GetUser(
	ctx context.Context,
	request adminapi.GetUserRequestObject,
) (adminapi.GetUserResponseObject, error) {
	u, err := dbGetUser(ctx, i.sqlClient, int64(request.UserID))
	if err != nil {
		if errors.Is(err, errNoUser) {
			return adminapi.GetUser404JSONResponse{
				NotFoundErrorJSONResponse: apiErrNotFound,
			}, nil
		}
		zerologr.Error(err, "Failed to get admin user")
		return adminapi.GetUser500JSONResponse{InternalErrorJSONResponse: apiErrInternal}, nil
	}

	groups, err := dbListGroupBindings(ctx, i.sqlClient, int64(u.Id))
	if err != nil {
		zerologr.Error(err, "Failed to list admin user group bindings")
		return adminapi.GetUser500JSONResponse{InternalErrorJSONResponse: apiErrInternal}, nil
	}

	apiGroups := make([]adminapi.Group, 0, len(groups))
	for _, b := range groups {
		apiGroups = append(apiGroups, adminapi.Group{Id: int(b.GroupID), Name: b.Name})
	}
	u.Groups = &apiGroups

	return adminapi.GetUser200JSONResponse(*u), nil
}

// UpdateUser implements [withExtensions].
func (i *impl) UpdateUser(
	ctx context.Context,
	request adminapi.UpdateUserRequestObject,
) (adminapi.UpdateUserResponseObject, error) {
	if request.Body.Username == nil {
		return adminapi.UpdateUser400JSONResponse{
			BadRequestErrorJSONResponse: adminapi.BadRequestErrorJSONResponse(
				makeGenAPIError("username is required"),
			),
		}, nil
	}

	if err := dbUpdateUser(
		ctx,
		i.sqlClient,
		int64(request.UserID),
		*request.Body.Username,
	); err != nil {
		if errors.Is(err, db.ErrUnique) {
			zerologr.Error(err, "Username conflict during update")
			return adminapi.UpdateUser409JSONResponse{
				ConflictErrorJSONResponse: apiErrConflict,
			}, nil
		}

		zerologr.Error(err, "Failed to update admin user")
		return adminapi.UpdateUser500JSONResponse{InternalErrorJSONResponse: apiErrInternal}, nil
	}

	return adminapi.UpdateUser204Response{}, nil
}

// DeleteUser implements [withExtensions].
func (i *impl) DeleteUser(
	ctx context.Context,
	request adminapi.DeleteUserRequestObject,
) (adminapi.DeleteUserResponseObject, error) {
	if _, err := dbGetUser(ctx, i.sqlClient, int64(request.UserID)); err != nil {
		if errors.Is(err, errNoUser) {
			return adminapi.DeleteUser404JSONResponse{
				NotFoundErrorJSONResponse: apiErrNotFound,
			}, nil
		}
		zerologr.Error(err, "Failed to check admin user before delete")
		return adminapi.DeleteUser500JSONResponse{InternalErrorJSONResponse: apiErrInternal}, nil
	}

	if err := dbDeleteUser(ctx, i.sqlClient, int64(request.UserID)); err != nil {
		zerologr.Error(err, "Failed to delete admin user")
		return adminapi.DeleteUser500JSONResponse{InternalErrorJSONResponse: apiErrInternal}, nil
	}

	return adminapi.DeleteUser204Response{}, nil
}

// ChangeUserPassword implements [withExtensions].
func (i *impl) ChangeUserPassword(
	ctx context.Context,
	request adminapi.ChangeUserPasswordRequestObject,
) (adminapi.ChangeUserPasswordResponseObject, error) {
	auth, err := dbGetUserAuth(ctx, i.sqlClient, int64(request.UserID))
	if err != nil {
		if errors.Is(err, errNoUser) {
			return adminapi.ChangeUserPassword404JSONResponse{
				NotFoundErrorJSONResponse: apiErrNotFound,
			}, nil
		}
		zerologr.Error(err, "Failed to get admin user auth for password change")
		return adminapi.ChangeUserPassword500JSONResponse{
			InternalErrorJSONResponse: apiErrInternal,
		}, nil
	}

	if !password.Match(auth.Salt, auth.HashedPassword, request.Body.OldPassword) {
		return adminapi.ChangeUserPassword401JSONResponse{
			UnauthorizedErrorJSONResponse: makeErrUnauthorized("Old password does not match."),
		}, nil
	}

	_, newSalt, newHashed := password.Make(request.Body.NewPassword)
	if err := dbUpdateUserPassword(
		ctx,
		i.sqlClient,
		int64(request.UserID),
		newSalt,
		newHashed,
	); err != nil {
		zerologr.Error(err, "Failed to update admin user password")
		return adminapi.ChangeUserPassword500JSONResponse{
			InternalErrorJSONResponse: apiErrInternal,
		}, nil
	}

	return adminapi.ChangeUserPassword204Response{}, nil
}

// UpdateUserGroups implements [withExtensions].
func (i *impl) UpdateUserGroups(
	ctx context.Context,
	request adminapi.UpdateUserGroupsRequestObject,
) (adminapi.UpdateUserGroupsResponseObject, error) {
	if _, err := dbGetUser(ctx, i.sqlClient, int64(request.UserID)); err != nil {
		if errors.Is(err, errNoUser) {
			return adminapi.UpdateUserGroups404JSONResponse{
				NotFoundErrorJSONResponse: apiErrNotFound,
			}, nil
		}
		zerologr.Error(err, "Failed to check admin user before group update")
		return adminapi.UpdateUserGroups500JSONResponse{
			InternalErrorJSONResponse: apiErrInternal,
		}, nil
	}

	if err := dbUpdateUserGroupBindings(
		ctx,
		i.sqlClient,
		int64(request.UserID),
		request.Body.GroupIDs,
	); err != nil {
		zerologr.Error(err, "Failed to update admin user group bindings")
		return adminapi.UpdateUserGroups500JSONResponse{
			InternalErrorJSONResponse: apiErrInternal,
		}, nil
	}

	return adminapi.UpdateUserGroups204Response{}, nil
}

// CreateGroup implements [withExtensions].
func (i *impl) CreateGroup(
	ctx context.Context,
	request adminapi.CreateGroupRequestObject,
) (adminapi.CreateGroupResponseObject, error) {
	id, err := dbCreateGroup(ctx, i.sqlClient, request.Body.Name)
	if err != nil {
		if errors.Is(err, db.ErrUnique) {
			zerologr.Error(err, "Group name conflict")
			return adminapi.CreateGroup409JSONResponse{
				ConflictErrorJSONResponse: apiErrConflict,
			}, nil
		}

		zerologr.Error(err, "Failed to create admin group")
		return adminapi.CreateGroup500JSONResponse{
			InternalErrorJSONResponse: apiErrInternal,
		}, nil
	}

	if err := dbSetGroupPermissions(ctx, i.sqlClient, id, request.Body.PermissionIDs); err != nil {
		zerologr.Error(err, "Failed to set permissions for admin group")
		return adminapi.CreateGroup500JSONResponse{
			InternalErrorJSONResponse: apiErrInternal,
		}, nil
	}

	perms, err := dbGetGroupPermissions(ctx, i.sqlClient, id)
	if err != nil {
		zerologr.Error(err, "Failed to fetch permissions for created admin group")
		return adminapi.CreateGroup500JSONResponse{
			InternalErrorJSONResponse: apiErrInternal,
		}, nil
	}

	return adminapi.CreateGroup201JSONResponse(
		adminapi.Group{Id: int(id), Name: request.Body.Name, Permissions: &perms},
	), nil
}

// GetGroups implements [withExtensions].
func (i *impl) GetGroups(
	ctx context.Context,
	_ adminapi.GetGroupsRequestObject,
) (adminapi.GetGroupsResponseObject, error) {
	groups, err := dbListGroups(ctx, i.sqlClient)
	if err != nil {
		zerologr.Error(err, "Failed to list admin groups")
		return adminapi.GetGroups500JSONResponse{InternalErrorJSONResponse: apiErrInternal}, nil
	}

	enriched := make([]adminapi.Group, 0, len(groups))
	for _, g := range groups {
		perms, err := dbGetGroupPermissions(ctx, i.sqlClient, int64(g.Id))
		if err != nil {
			zerologr.Error(err, "Failed to fetch permissions for admin group")
			return adminapi.GetGroups500JSONResponse{InternalErrorJSONResponse: apiErrInternal}, nil
		}
		g.Permissions = &perms
		enriched = append(enriched, g)
	}

	return adminapi.GetGroups200JSONResponse(enriched), nil
}

// GetGroup implements [withExtensions].
func (i *impl) GetGroup(
	ctx context.Context,
	request adminapi.GetGroupRequestObject,
) (adminapi.GetGroupResponseObject, error) {
	g, err := dbGetGroup(ctx, i.sqlClient, int64(request.GroupID))
	if err != nil {
		if errors.Is(err, errNoGroup) {
			return adminapi.GetGroup404JSONResponse{
				NotFoundErrorJSONResponse: apiErrNotFound,
			}, nil
		}
		zerologr.Error(err, "Failed to get admin group")
		return adminapi.GetGroup500JSONResponse{InternalErrorJSONResponse: apiErrInternal}, nil
	}

	perms, err := dbGetGroupPermissions(ctx, i.sqlClient, int64(request.GroupID))
	if err != nil {
		zerologr.Error(err, "Failed to fetch permissions for admin group")
		return adminapi.GetGroup500JSONResponse{InternalErrorJSONResponse: apiErrInternal}, nil
	}
	g.Permissions = &perms

	return adminapi.GetGroup200JSONResponse(*g), nil
}

// UpdateGroup implements [withExtensions].
func (i *impl) UpdateGroup(
	ctx context.Context,
	request adminapi.UpdateGroupRequestObject,
) (adminapi.UpdateGroupResponseObject, error) {
	if _, err := dbGetGroup(ctx, i.sqlClient, int64(request.GroupID)); err != nil {
		if errors.Is(err, errNoGroup) {
			return adminapi.UpdateGroup404JSONResponse{
				NotFoundErrorJSONResponse: apiErrNotFound,
			}, nil
		}

		zerologr.Error(err, "Failed to check admin group before update")
		return adminapi.UpdateGroup500JSONResponse{
			InternalErrorJSONResponse: apiErrInternal,
		}, nil
	}

	if err := dbUpdateGroup(
		ctx,
		i.sqlClient,
		int64(request.GroupID),
		request.Body.Name,
	); err != nil {
		if errors.Is(err, db.ErrUnique) {
			zerologr.Error(err, "Group name conflict during update")
			return adminapi.UpdateGroup409JSONResponse{
				ConflictErrorJSONResponse: apiErrConflict,
			}, nil
		}

		zerologr.Error(err, "Failed to update admin group")
		return adminapi.UpdateGroup500JSONResponse{
			InternalErrorJSONResponse: apiErrInternal,
		}, nil
	}

	if err := dbSetGroupPermissions(ctx, i.sqlClient, int64(request.GroupID), request.Body.PermissionIDs); err != nil {
		zerologr.Error(err, "Failed to update permissions for admin group")
		return adminapi.UpdateGroup500JSONResponse{
			InternalErrorJSONResponse: apiErrInternal,
		}, nil
	}

	return adminapi.UpdateGroup204Response{}, nil
}

// DeleteGroup implements [withExtensions].
func (i *impl) DeleteGroup(
	ctx context.Context,
	request adminapi.DeleteGroupRequestObject,
) (adminapi.DeleteGroupResponseObject, error) {
	if _, err := dbGetGroup(ctx, i.sqlClient, int64(request.GroupID)); err != nil {
		if errors.Is(err, errNoGroup) {
			return adminapi.DeleteGroup404JSONResponse{
				NotFoundErrorJSONResponse: apiErrNotFound,
			}, nil
		}
		zerologr.Error(err, "Failed to check admin group before delete")
		return adminapi.DeleteGroup500JSONResponse{
			InternalErrorJSONResponse: apiErrInternal,
		}, nil
	}

	if err := dbDeleteGroup(ctx, i.sqlClient, int64(request.GroupID)); err != nil {
		zerologr.Error(err, "Failed to delete admin group")
		return adminapi.DeleteGroup500JSONResponse{
			InternalErrorJSONResponse: apiErrInternal,
		}, nil
	}

	return adminapi.DeleteGroup204Response{}, nil
}

// bootstrapSuperuser checks if a super user exists and if not, creates one with the provided credentials.
// This is to allow bootstrapping of the first super user. Subsequent calls to this function will not have any effect.
// This is to prevent re-provisioning of the super-user, potentially allowing an attacker to reset powerful credentials.
func (i *impl) bootstrapSuperuser(clientID, clientSecret string) {
	// check if a super user already exists.
	rows, err := i.sqlClient.Query(context.Background(), selectSuperuser)
	if err != nil {
		panic(err)
	}
	defer rows.Close()

	// No row found, check if due to an error.
	if !rows.Next() {
		// Error, panic.
		if err := rows.Err(); err != nil {
			panic(err)
		}

		// No super user exists, create one with the provided credentials.
		_, salt, hashedPassword := password.Make(clientSecret)
		if _, err := i.sqlClient.Exec(
			context.TODO(),
			insertSuperuser,
			sql.NamedArg{Name: "name", Value: clientID},
			sql.NamedArg{Name: "salt", Value: salt},
			sql.NamedArg{Name: "hashed_password", Value: hashedPassword},
		); err != nil {
			panic(err)
		}
	}

	// no-op, a row existed
}
