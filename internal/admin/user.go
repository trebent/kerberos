// nolint:revive // temporary
package admin

import (
	"context"
	"database/sql"
	"time"

	"github.com/google/uuid"
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
	superuser, err := i.querySuperuser()
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

// ChangeUserPassword implements [withExtensions].
func (i *impl) ChangeUserPassword(
	ctx context.Context,
	request adminapi.ChangeUserPasswordRequestObject,
) (adminapi.ChangeUserPasswordResponseObject, error) {
	return nil, apierror.APIErrUnimplemented
}

// CreateGroup implements [withExtensions].
func (i *impl) CreateGroup(
	ctx context.Context,
	request adminapi.CreateGroupRequestObject,
) (adminapi.CreateGroupResponseObject, error) {
	return nil, apierror.APIErrUnimplemented
}

// CreateUser implements [withExtensions].
func (i *impl) CreateUser(
	ctx context.Context,
	request adminapi.CreateUserRequestObject,
) (adminapi.CreateUserResponseObject, error) {
	return nil, apierror.APIErrUnimplemented
}

// DeleteGroup implements [withExtensions].
func (i *impl) DeleteGroup(
	ctx context.Context,
	request adminapi.DeleteGroupRequestObject,
) (adminapi.DeleteGroupResponseObject, error) {
	return nil, apierror.APIErrUnimplemented
}

// DeleteUser implements [withExtensions].
func (i *impl) DeleteUser(
	ctx context.Context,
	request adminapi.DeleteUserRequestObject,
) (adminapi.DeleteUserResponseObject, error) {
	return nil, apierror.APIErrUnimplemented
}

// GetGroup implements [withExtensions].
func (i *impl) GetGroup(
	ctx context.Context,
	request adminapi.GetGroupRequestObject,
) (adminapi.GetGroupResponseObject, error) {
	return nil, apierror.APIErrUnimplemented
}

// GetGroups implements [withExtensions].
func (i *impl) GetGroups(
	ctx context.Context,
	request adminapi.GetGroupsRequestObject,
) (adminapi.GetGroupsResponseObject, error) {
	return nil, apierror.APIErrUnimplemented
}

// GetUser implements [withExtensions].
func (i *impl) GetUser(
	ctx context.Context,
	request adminapi.GetUserRequestObject,
) (adminapi.GetUserResponseObject, error) {
	return nil, apierror.APIErrUnimplemented
}

// GetUsers implements [withExtensions].
func (i *impl) GetUsers(
	ctx context.Context,
	request adminapi.GetUsersRequestObject,
) (adminapi.GetUsersResponseObject, error) {
	return nil, apierror.APIErrUnimplemented
}

// Login implements [withExtensions].
func (i *impl) Login(
	ctx context.Context,
	request adminapi.LoginRequestObject,
) (adminapi.LoginResponseObject, error) {
	return nil, apierror.APIErrUnimplemented
}

// Logout implements [withExtensions].
func (i *impl) Logout(
	ctx context.Context,
	request adminapi.LogoutRequestObject,
) (adminapi.LogoutResponseObject, error) {
	return nil, apierror.APIErrUnimplemented
}

// UpdateGroup implements [withExtensions].
func (i *impl) UpdateGroup(
	ctx context.Context,
	request adminapi.UpdateGroupRequestObject,
) (adminapi.UpdateGroupResponseObject, error) {
	return nil, apierror.APIErrUnimplemented
}

// UpdateUser implements [withExtensions].
func (i *impl) UpdateUser(
	ctx context.Context,
	request adminapi.UpdateUserRequestObject,
) (adminapi.UpdateUserResponseObject, error) {
	return nil, apierror.APIErrUnimplemented
}

// UpdateUserGroups implements [withExtensions].
func (i *impl) UpdateUserGroups(
	ctx context.Context,
	request adminapi.UpdateUserGroupsRequestObject,
) (adminapi.UpdateUserGroupsResponseObject, error) {
	return nil, apierror.APIErrUnimplemented
}

// bootstrapSuperuser checks if a super user exists and if not, creates one with the provided credentials.
// This is to allow bootstrapping of the first super user. Subsequent calls to this function will not have any effect.
// This is to prevent re-provisioning of the super-user, potentially allowing an attacker to reset powerful credentials.
func (i *impl) bootstrapSuperuser(clientID, clientSecret string) {
	// check if a super user already exists.
	rows, err := i.sqlClient.Query(context.Background(), querySuperuser)
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
