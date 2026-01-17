package api

import (
	"context"
	"database/sql"
	"fmt"
	"slices"
	"time"

	"github.com/google/uuid"
	"github.com/trebent/kerberos/internal/apierror"
	"github.com/trebent/kerberos/internal/auth/util"
	"github.com/trebent/kerberos/internal/db"
	"github.com/trebent/zerologr"
)

//go:generate go tool oapi-codegen -config ./config.yaml -o ./gen.go ../../../../../openapi/basic_auth.yaml

type impl struct {
	db db.SQLClient
}

var (
	_ StrictServerInterface = (*impl)(nil)

	// Organisations.
	queryCreateOrg = "INSERT INTO organisations (name) VALUES(@name);"
	queryDeleteOrg = "DELETE FROM organisations WHERE id = @orgID;"
	queryGetOrg    = "SELECT id, name FROM organisations WHERE id = @orgID;"
	queryListOrgs  = "SELECT id, name FROM organisations;"
	queryUpdateOrg = "UPDATE organisations SET name = @name WHERE id = @orgID;"

	// Groups.
	queryCreateGroup = "INSERT INTO groups (name, organisation_id) VALUES(@name, @orgID);"
	queryDeleteGroup = "DELETE FROM groups WHERE organisation_id = @orgID AND group_id = @groupID;"
	queryGetGroup    = "SELECT id, name FROM groups WHERE id = @groupID AND organisation_id = @orgID;"
	queryListGroups  = "SELECT id, name FROM groups WHERE organisation_id = @orgID;"
	queryUpdateGroup = "UPDATE groups SET name = @name WHERE id = @groupID AND organisation_id = @orgID;"

	// Users.
	queryCreateUser  = "INSERT INTO users (name, salt, hashed_password, organisation_id, administrator) VALUES(@name, @salt, @hashed_password, @orgID, @isAdmin);"
	queryDeleteUser  = "DELETE FROM users WHERE id = @userID AND organisation_id = @orgID;"
	queryGetUser     = "SELECT id, name FROM users WHERE id = @userID AND organisation_id = @orgID;"
	queryGetFullUser = "SELECT id, name, salt, hashed_password FROM users WHERE id = @userID AND organisation_id = @orgID;"
	queryListUsers   = "SELECT id, name FROM users WHERE organisation_id = @orgID;"
	queryUpdateUser  = "UPDATE users SET name = @name WHERE id = @userID AND organisation_id = @orgID;"
	//nolint:gosec // not a password
	queryUpdateUserPassword = "UPDATE users SET salt = @salt, hashed_password = @hashed_password WHERE id = @id;"
	queryLoginLookup        = "SELECT id, name, salt, hashed_password, organisation_id FROM users WHERE organisation_id = @orgID AND name = @username;"

	// Group bindings.
	queryListUserGroups     = "SELECT name FROM groups WHERE id IN (SELECT group_id FROM group_bindings WHERE user_id = @userID) AND organisation_id = @orgID;"
	queryListGroupBindings  = "SELECT g.id, g.name FROM group_bindings gb INNER JOIN groups g ON gb.group_id = g.id WHERE user_id = @userID AND organisation_id = @orgID;"
	queryDeleteGroupBinding = "DELETE FROM group_bindings WHERE user_id = @userID AND group_id = @groupID;"
	queryCreateGroupBinding = "INSERT INTO group_bindings (user_id, group_id) VALUES (@userID, (SELECT id FROM groups WHERE organisation_id = @orgID AND name = @groupName));"

	// Sessions.
	queryCreateSession      = "INSERT INTO sessions (user_id, organisation_id, session_id, expires) VALUES(@userID, @orgID, @session, @expires);"
	queryGetSession         = "SELECT s.user_id, s.organisation_id, u.administrator, u.super_user, s.expires FROM sessions s INNER JOIN users u ON s.user_id = u.id WHERE session_id = @sessionID;"
	queryDeleteUserSessions = "DELETE FROM sessions WHERE organisation_id = @orgID AND user_id = @userID;"

	GenErrInternal = APIErrorResponse{Errors: []string{apierror.ErrInternal.Error()}}
)

const sessionExpiry = 15 * time.Minute

func makeGenAPIError(msg string) APIErrorResponse {
	return APIErrorResponse{Errors: []string{msg}}
}

func NewSSI(db db.SQLClient) StrictServerInterface {
	return &impl{db}
}

// Login implements [StrictServerInterface].
func (i *impl) Login(ctx context.Context, req LoginRequestObject) (LoginResponseObject, error) {
	rows, err := i.db.Query(
		ctx,
		queryLoginLookup,
		sql.NamedArg{Name: "orgID", Value: req.OrgID},
		sql.NamedArg{Name: "username", Value: req.Body.Username},
	)
	if err != nil {
		zerologr.Error(err, "Failed to query for a user")
		return Login500JSONResponse(GenErrInternal), nil
	}

	if !rows.Next() {
		if err := rows.Err(); err != nil {
			zerologr.Error(err, "Failed to prepare rows for scanning")
		}

		return Login401JSONResponse(makeGenAPIError("Login failed.")), nil
	}

	id := 0
	salt := ""
	storedHashed := ""
	organisationID := 0
	err = rows.Scan(&id, new(string), &salt, &storedHashed, &organisationID)
	//nolint:sqlclosecheck // won't help here
	_ = rows.Close()
	if err != nil {
		zerologr.Error(err, "Failed to scan for a matching user row")
		return Login500JSONResponse(GenErrInternal), nil
	}

	if !util.PasswordMatch(salt, storedHashed, req.Body.Password) {
		zerologr.Info("User login failed due to password mismatch")
		return Login401JSONResponse(makeGenAPIError("Login failed.")), nil
	}
	zerologr.V(10).Info("User has logged in successfully", "username", req.Body.Username)

	sessionID := uuid.NewString()
	_, err = i.db.Exec(
		ctx,
		queryCreateSession,
		sql.NamedArg{Name: "userID", Value: id},
		sql.NamedArg{Name: "orgID", Value: organisationID},
		sql.NamedArg{Name: "session", Value: sessionID},
		sql.NamedArg{Name: "expires", Value: time.Now().Add(sessionExpiry).UnixMilli()},
	)
	if err != nil {
		zerologr.Error(err, "Failed to store new session ID")
		return Login500JSONResponse(GenErrInternal), nil
	}

	return Login204Response{
		Headers: Login204ResponseHeaders{
			XKrbSession: sessionID,
		},
	}, nil
}

// Logout implements [StrictServerInterface].
func (i *impl) Logout(
	ctx context.Context,
	req LogoutRequestObject,
) (LogoutResponseObject, error) {
	userID := userFromContext(ctx)
	if _, err := i.db.Exec(
		ctx,
		queryDeleteUserSessions,
		sql.NamedArg{Name: "orgID", Value: req.OrgID},
		sql.NamedArg{Name: "userID", Value: userID},
	); err != nil {
		zerologr.Error(err, "Failed to clear user sessions")
		return Logout500JSONResponse(GenErrInternal), nil
	}

	return Logout204Response{}, nil
}

// ChangePassword implements [StrictServerInterface].
func (i *impl) ChangePassword(
	ctx context.Context,
	req ChangePasswordRequestObject,
) (ChangePasswordResponseObject, error) {
	rows, err := i.db.Query(
		ctx,
		queryGetFullUser,
		sql.NamedArg{Name: "orgID", Value: req.OrgID},
		sql.NamedArg{Name: "userID", Value: req.UserID},
	)
	if err != nil {
		zerologr.Error(err, "Failed to get user from session")
		return ChangePassword500JSONResponse(GenErrInternal), nil
	}

	if !rows.Next() {
		if err := rows.Err(); err != nil {
			zerologr.Error(err, "Failed to scan rows")
			return ChangePassword500JSONResponse(GenErrInternal), nil
		}

		return ChangePassword401JSONResponse(
			makeGenAPIError("Failed to change user password."),
		), nil
	}

	salt := ""
	hashedPassword := ""
	err = rows.Scan(new(int64), new(string), &salt, &hashedPassword)
	//nolint:sqlclosecheck // won't help here
	_ = rows.Close()
	if err != nil {
		zerologr.Error(err, "Failed to scan row")
		return ChangePassword500JSONResponse(GenErrInternal), nil
	}

	if !util.PasswordMatch(salt, hashedPassword, req.Body.OldPassword) {
		zerologr.Error(err, "Mismatched old password")
		return ChangePassword401JSONResponse(
			makeGenAPIError("Failed to change user password."),
		), nil
	}

	_, newSalt, newHashedPassword := util.MakePassword(req.Body.Password)
	_, err = i.db.Exec(
		ctx,
		queryUpdateUserPassword,
		sql.NamedArg{Name: "salt", Value: newSalt},
		sql.NamedArg{Name: "hashed_password", Value: newHashedPassword},
		sql.NamedArg{Name: "id", Value: req.UserID},
	)
	if err != nil {
		zerologr.Error(err, "Failed to update user password")
		return ChangePassword500JSONResponse(GenErrInternal), nil
	}

	return ChangePassword204Response{}, nil
}

// CreateGroup implements [StrictServerInterface].
func (i *impl) CreateGroup(
	ctx context.Context,
	req CreateGroupRequestObject,
) (CreateGroupResponseObject, error) {
	res, err := i.db.Exec(
		ctx,
		queryCreateGroup,
		sql.NamedArg{Name: "name", Value: req.Body.Name},
		sql.NamedArg{Name: "orgID", Value: req.OrgID},
	)
	if err != nil {
		zerologr.Error(err, "Failed to insert group")
		return CreateGroup500JSONResponse(GenErrInternal), nil
	}

	id, _ := res.LastInsertId()
	return CreateGroup201JSONResponse{
		Id:   id,
		Name: req.Body.Name,
	}, nil
}

// CreateOrganisation implements [StrictServerInterface].
func (i *impl) CreateOrganisation(
	ctx context.Context,
	req CreateOrganisationRequestObject,
) (CreateOrganisationResponseObject, error) {
	tx, err := i.db.Begin(ctx)
	if err != nil {
		zerologr.Error(err, "Failed to start transaction")
		return CreateOrganisation500JSONResponse(GenErrInternal), nil
	}
	//nolint:errcheck // no reason to
	defer tx.Rollback() // Just in case

	// Create an organisation.
	res, err := tx.Exec(ctx, queryCreateOrg, sql.NamedArg{Name: "name", Value: req.Body.Name})
	if err != nil {
		zerologr.Error(err, "Failed to create org")
		return CreateOrganisation500JSONResponse(GenErrInternal), nil
	}
	id, _ := res.LastInsertId()
	zerologr.Info(fmt.Sprintf("Created organisation with id %d", id))

	// Create an admin user for the organisation.
	adminUsername := fmt.Sprintf("%s-%s", "admin", req.Body.Name)

	adminPassword, salt, hashedAdminPassword := util.MakePassword("")
	res, err = tx.Exec(
		ctx,
		queryCreateUser,
		sql.NamedArg{Name: "name", Value: adminUsername},
		sql.NamedArg{Name: "salt", Value: salt},
		sql.NamedArg{Name: "hashed_password", Value: hashedAdminPassword},
		sql.NamedArg{Name: "orgID", Value: id},
		sql.NamedArg{Name: "isAdmin", Value: true},
	)
	if err != nil {
		zerologr.Error(err, "Failed to create admin user")
		return CreateOrganisation500JSONResponse(GenErrInternal), nil
	}

	err = tx.Commit()
	if err != nil {
		zerologr.Error(err, "Failed to commit transaction")
		return CreateOrganisation500JSONResponse(GenErrInternal), nil
	}

	userID, _ := res.LastInsertId()
	return CreateOrganisation201JSONResponse{
		Id:            id,
		Name:          req.Body.Name,
		AdminUserId:   userID,
		AdminPassword: adminPassword,
		AdminUsername: adminUsername,
	}, nil
}

// CreateUser implements [StrictServerInterface].
func (i *impl) CreateUser(
	ctx context.Context,
	req CreateUserRequestObject,
) (CreateUserResponseObject, error) {
	_, salt, hashedPassword := util.MakePassword(req.Body.Password)
	res, err := i.db.Exec(
		ctx,
		queryCreateUser,
		sql.NamedArg{Name: "name", Value: req.Body.Name},
		sql.NamedArg{Name: "salt", Value: salt},
		sql.NamedArg{Name: "hashed_password", Value: hashedPassword},
		sql.NamedArg{Name: "orgID", Value: req.OrgID},
		sql.NamedArg{Name: "isAdmin", Value: false},
	)
	if err != nil {
		zerologr.Error(err, "Failed to insert new user")
		return CreateUser500JSONResponse(GenErrInternal), nil
	}

	id, _ := res.LastInsertId()
	return CreateUser201JSONResponse{
		Id:   id,
		Name: req.Body.Name,
	}, nil
}

// DeleteGroup implements [StrictServerInterface].
func (i *impl) DeleteGroup(
	ctx context.Context,
	req DeleteGroupRequestObject,
) (DeleteGroupResponseObject, error) {
	_, err := i.db.Exec(
		ctx,
		queryDeleteGroup,
		sql.NamedArg{Name: "orgID", Value: req.OrgID},
		sql.NamedArg{Name: "groupID", Value: req.GroupID},
	)
	if err != nil {
		zerologr.Error(err, "Failed to delete group")
		return DeleteGroup500JSONResponse(GenErrInternal), nil
	}

	return DeleteGroup204Response{}, nil
}

// DeleteOrganisation implements [StrictServerInterface].
func (i *impl) DeleteOrganisation(
	ctx context.Context,
	req DeleteOrganisationRequestObject,
) (DeleteOrganisationResponseObject, error) {
	_, err := i.db.Exec(
		ctx,
		queryDeleteOrg,
		sql.NamedArg{Name: "orgID", Value: req.OrgID},
	)
	if err != nil {
		zerologr.Error(err, "Failed to delete org")
		return DeleteOrganisation500JSONResponse(GenErrInternal), nil
	}

	return DeleteOrganisation204Response{}, nil
}

// DeleteUser implements [StrictServerInterface].
func (i *impl) DeleteUser(
	ctx context.Context,
	req DeleteUserRequestObject,
) (DeleteUserResponseObject, error) {
	_, err := i.db.Exec(
		ctx,
		queryDeleteUser,
		sql.NamedArg{Name: "orgID", Value: req.OrgID},
		sql.NamedArg{Name: "userID", Value: req.UserID},
	)
	if err != nil {
		zerologr.Error(err, "Failed to delete user")
		return DeleteUser500JSONResponse(GenErrInternal), nil
	}

	return DeleteUser204Response{}, nil
}

// GetGroup implements [StrictServerInterface].
//
//nolint:dupl // welp
func (i *impl) GetGroup(
	ctx context.Context,
	req GetGroupRequestObject,
) (GetGroupResponseObject, error) {
	rows, err := i.db.Query(
		ctx,
		queryGetGroup,
		sql.NamedArg{Name: "groupID", Value: req.GroupID},
		sql.NamedArg{Name: "orgID", Value: req.OrgID},
	)
	if err != nil {
		zerologr.Error(err, "Failed to query groups")
		return GetGroup500JSONResponse(GenErrInternal), nil
	}

	if !rows.Next() {
		if err := rows.Err(); err != nil {
			zerologr.Error(err, "Failed to scan next row")
			return GetGroup500JSONResponse(GenErrInternal), nil
		}

		return GetGroup404Response{}, nil
	}

	var (
		id   int64
		name string
	)
	err = rows.Scan(&id, &name)
	//nolint:sqlclosecheck // won't help here
	_ = rows.Close()
	if err != nil {
		zerologr.Error(err, "Failed to scan row")
		return GetGroup500JSONResponse(GenErrInternal), nil
	}

	return GetGroup200JSONResponse{
		Id:   id,
		Name: name,
	}, nil
}

// GetOrganisation implements [StrictServerInterface].
//
//nolint:dupl // welp
func (i *impl) GetOrganisation(
	ctx context.Context,
	req GetOrganisationRequestObject,
) (GetOrganisationResponseObject, error) {
	rows, err := i.db.Query(
		ctx,
		queryGetOrg,
		sql.NamedArg{Name: "orgID", Value: req.OrgID},
	)
	if err != nil {
		zerologr.Error(err, "Failed to query organisations")
		return GetOrganisation500JSONResponse(GenErrInternal), nil
	}

	if !rows.Next() {
		if err := rows.Err(); err != nil {
			zerologr.Error(err, "Failed to scan next row")
			return GetOrganisation500JSONResponse(GenErrInternal), nil
		}

		return GetOrganisation404Response{}, nil
	}

	var (
		id   int64
		name string
	)
	err = rows.Scan(&id, &name)
	//nolint:sqlclosecheck // won't help here
	_ = rows.Close()
	if err != nil {
		zerologr.Error(err, "Failed to scan row")
		return GetOrganisation500JSONResponse(GenErrInternal), nil
	}

	return GetOrganisation200JSONResponse{
		Id:   id,
		Name: name,
	}, nil
}

// GetUser implements [StrictServerInterface].
//
//nolint:dupl // welp
func (i *impl) GetUser(
	ctx context.Context,
	req GetUserRequestObject,
) (GetUserResponseObject, error) {
	rows, err := i.db.Query(
		ctx,
		queryGetUser,
		sql.NamedArg{Name: "orgID", Value: req.OrgID},
		sql.NamedArg{Name: "userID", Value: req.UserID},
	)
	if err != nil {
		zerologr.Error(err, "Failed to query users")
		return GetUser500JSONResponse(GenErrInternal), nil
	}

	if !rows.Next() {
		if err := rows.Err(); err != nil {
			zerologr.Error(err, "Failed to scan next row")
			return GetUser500JSONResponse(GenErrInternal), nil
		}

		return GetUser404Response{}, nil
	}

	var (
		id   int64
		name string
	)
	err = rows.Scan(&id, &name)
	//nolint:sqlclosecheck // won't help here
	_ = rows.Close()
	if err != nil {
		zerologr.Error(err, "Failed to scan row")
		return GetUser500JSONResponse(GenErrInternal), nil
	}

	return GetUser200JSONResponse{
		Id:   id,
		Name: name,
	}, nil
}

// GetUserGroups implements [StrictServerInterface].
func (i *impl) GetUserGroups(
	ctx context.Context,
	req GetUserGroupsRequestObject,
) (GetUserGroupsResponseObject, error) {
	rows, err := i.db.Query(
		ctx,
		queryListUserGroups,
		sql.NamedArg{Name: "orgID", Value: req.OrgID},
		sql.NamedArg{Name: "userID", Value: req.UserID},
	)
	if err != nil {
		zerologr.Error(err, "Failed to query user groups")
		return GetUserGroups500JSONResponse(GenErrInternal), nil
	}
	// Fine to defer since we're iterating, not just doing one scan.
	defer rows.Close()

	userGroups := make([]string, 0)
	for {
		if !rows.Next() {
			if err := rows.Err(); err != nil {
				zerologr.Error(err, "Failed to scan next row")
				return GetUserGroups500JSONResponse(GenErrInternal), nil
			}

			return GetUserGroups200JSONResponse(userGroups), nil
		}

		groupName := ""
		if err = rows.Scan(&groupName); err != nil {
			zerologr.Error(err, "Failed to scan row")
			return GetUserGroups500JSONResponse(GenErrInternal), nil
		}

		userGroups = append(userGroups, groupName)
	}
}

// ListGroups implements [StrictServerInterface].
//
//nolint:dupl // welp
func (i *impl) ListGroups(
	ctx context.Context,
	req ListGroupsRequestObject,
) (ListGroupsResponseObject, error) {
	rows, err := i.db.Query(
		ctx,
		queryListGroups,
		sql.NamedArg{Name: "orgID", Value: req.OrgID},
	)
	if err != nil {
		zerologr.Error(err, "Failed to query groups")
		return ListGroups500JSONResponse(GenErrInternal), nil
	}
	// Fine to defer since we're iterating, not just doing one scan.
	defer rows.Close()

	groups := make([]Group, 0)
	for {
		if !rows.Next() {
			if err := rows.Err(); err != nil {
				zerologr.Error(err, "Failed to scan next row")
				return ListGroups500JSONResponse(GenErrInternal), nil
			}

			return ListGroups200JSONResponse(groups), nil
		}

		var (
			id   int64
			name string
		)
		if err = rows.Scan(&id, &name); err != nil {
			zerologr.Error(err, "Failed to scan row")
			return ListGroups500JSONResponse(GenErrInternal), nil
		}

		groups = append(groups, Group{Id: id, Name: name})
	}
}

// ListOrganisations implements [StrictServerInterface].
func (i *impl) ListOrganisations(
	ctx context.Context,
	_ ListOrganisationsRequestObject,
) (ListOrganisationsResponseObject, error) {
	rows, err := i.db.Query(
		ctx,
		queryListOrgs,
	)
	if err != nil {
		zerologr.Error(err, "Failed to query organisations")
		return ListOrganisations500JSONResponse(GenErrInternal), nil
	}
	// Fine to defer since we're iterating, not just doing one scan.
	defer rows.Close()

	orgs := make([]Organisation, 0)
	for {
		if !rows.Next() {
			if err := rows.Err(); err != nil {
				zerologr.Error(err, "Failed to scan next row")
				return ListOrganisations500JSONResponse(GenErrInternal), nil
			}

			return ListOrganisations200JSONResponse(orgs), nil
		}

		var (
			id   int64
			name string
		)
		if err = rows.Scan(&id, &name); err != nil {
			zerologr.Error(err, "Failed to scan row")
			return ListOrganisations500JSONResponse(GenErrInternal), nil
		}

		orgs = append(orgs, Organisation{Id: id, Name: name})
	}
}

// ListUsers implements [StrictServerInterface].
//
//nolint:dupl // welp
func (i *impl) ListUsers(
	ctx context.Context,
	req ListUsersRequestObject,
) (ListUsersResponseObject, error) {
	rows, err := i.db.Query(
		ctx,
		queryListUsers,
		sql.NamedArg{Name: "orgID", Value: req.OrgID},
	)
	if err != nil {
		zerologr.Error(err, "Failed to query users")
		return ListUsers500JSONResponse(GenErrInternal), nil
	}
	// Fine to defer since we're iterating, not just doing one scan.
	defer rows.Close()

	users := make([]User, 0)
	for {
		if !rows.Next() {
			if err := rows.Err(); err != nil {
				zerologr.Error(err, "Failed to scan next row")
				return ListUsers500JSONResponse(GenErrInternal), nil
			}

			return ListUsers200JSONResponse(users), nil
		}

		var (
			id   int64
			name string
		)
		if err = rows.Scan(&id, &name); err != nil {
			zerologr.Error(err, "Failed to scan row")
			return ListUsers500JSONResponse(GenErrInternal), nil
		}

		users = append(users, User{Id: id, Name: name})
	}
}

// UpdateGroup implements [StrictServerInterface].
func (i *impl) UpdateGroup(
	ctx context.Context,
	req UpdateGroupRequestObject,
) (UpdateGroupResponseObject, error) {
	_, err := i.db.Exec(
		ctx,
		queryUpdateGroup,
		sql.NamedArg{Name: "name", Value: req.Body.Name},
		sql.NamedArg{Name: "groupID", Value: req.GroupID},
		sql.NamedArg{Name: "orgID", Value: req.OrgID},
	)
	if err != nil {
		zerologr.Error(err, "Failed to update group")
		return UpdateGroup500JSONResponse(GenErrInternal), nil
	}

	return UpdateGroup200JSONResponse{
		Id:   req.GroupID,
		Name: req.Body.Name,
	}, nil
}

// UpdateOrganisation implements [StrictServerInterface].
func (i *impl) UpdateOrganisation(
	ctx context.Context,
	req UpdateOrganisationRequestObject,
) (UpdateOrganisationResponseObject, error) {
	_, err := i.db.Exec(
		ctx,
		queryUpdateOrg,
		sql.NamedArg{Name: "name", Value: req.Body.Name},
		sql.NamedArg{Name: "orgID", Value: req.OrgID},
	)
	if err != nil {
		zerologr.Error(err, "Failed to update organisation")
		return UpdateOrganisation500JSONResponse(GenErrInternal), nil
	}

	return UpdateOrganisation200JSONResponse{
		Id:   req.OrgID,
		Name: req.Body.Name,
	}, nil
}

// UpdateUser implements [StrictServerInterface].
func (i *impl) UpdateUser(
	ctx context.Context,
	req UpdateUserRequestObject,
) (UpdateUserResponseObject, error) {
	_, err := i.db.Exec(
		ctx,
		queryUpdateUser,
		sql.NamedArg{Name: "name", Value: req.Body.Name},
		sql.NamedArg{Name: "orgID", Value: req.OrgID},
		sql.NamedArg{Name: "userID", Value: req.UserID},
	)
	if err != nil {
		zerologr.Error(err, "Failed to update user")
		return UpdateUser500JSONResponse(GenErrInternal), nil
	}

	return UpdateUser200JSONResponse{
		Id:   req.UserID,
		Name: req.Body.Name,
	}, nil
}

// UpdateUserGroups implements [StrictServerInterface].
//
//nolint:gocognit // welp
func (i *impl) UpdateUserGroups(
	ctx context.Context,
	req UpdateUserGroupsRequestObject,
) (UpdateUserGroupsResponseObject, error) {
	rows, err := i.db.Query(
		ctx,
		queryListGroupBindings,
		sql.NamedArg{Name: "orgID", Value: req.OrgID},
		sql.NamedArg{Name: "userID", Value: req.UserID},
	)
	if err != nil {
		zerologr.Error(err, "Failed to query group bindings")
		return UpdateUserGroups500JSONResponse(GenErrInternal), nil
	}
	defer rows.Close()

	type internalGroupBinding struct {
		groupID int64
		name    string
	}
	bindings := make([]*internalGroupBinding, 0)
	for {
		if !rows.Next() {
			if err := rows.Err(); err != nil {
				zerologr.Error(err, "Failed to scan next row")
				return UpdateUserGroups500JSONResponse(GenErrInternal), nil
			}

			break
		}

		binding := &internalGroupBinding{}
		if err = rows.Scan(&binding.groupID, &binding.name); err != nil {
			zerologr.Error(err, "Failed to scan row")
			return UpdateUserGroups500JSONResponse(GenErrInternal), nil
		}

		bindings = append(bindings, binding)
	}

	toDelete := make([]*internalGroupBinding, 0)
	for _, existingBinding := range bindings {
		if !slices.Contains(*req.Body, existingBinding.name) {
			toDelete = append(toDelete, existingBinding)
		}
	}

	tx, err := i.db.Begin(ctx)
	if err != nil {
		zerologr.Error(err, "Failed to start transaction")
		return UpdateUserGroups500JSONResponse(GenErrInternal), nil
	}
	//nolint:errcheck // no reason to
	defer tx.Rollback()

	for _, bindingToDelete := range toDelete {
		_, err := tx.Exec(
			ctx,
			queryDeleteGroupBinding,
			sql.NamedArg{Name: "userID", Value: req.UserID},
			sql.NamedArg{Name: "groupID", Value: bindingToDelete.groupID},
		)
		if err != nil {
			zerologr.Error(err, "Failed to run group binding deletion")
			return UpdateUserGroups500JSONResponse(GenErrInternal), nil
		}

		bindings = slices.DeleteFunc(
			bindings,
			func(binding *internalGroupBinding) bool { return binding.name == bindingToDelete.name },
		)
	}

	for _, requestBinding := range *req.Body {
		if !slices.ContainsFunc(
			bindings,
			func(binding *internalGroupBinding) bool { return binding.name == requestBinding },
		) {
			_, err = tx.Exec(
				ctx,
				queryCreateGroupBinding,
				sql.NamedArg{Name: "userID", Value: req.UserID},
				sql.NamedArg{Name: "orgID", Value: req.OrgID},
				sql.NamedArg{Name: "groupName", Value: requestBinding},
			)
			if err != nil {
				zerologr.Error(err, "Failed to insert new binding")
				return UpdateUserGroups500JSONResponse(GenErrInternal), nil
			}
		}
	}

	if err := tx.Commit(); err != nil {
		zerologr.Error(err, "Failed to commit transaction")
		return UpdateUserGroups500JSONResponse(GenErrInternal), nil
	}

	return UpdateUserGroups200JSONResponse(*req.Body), nil
}
