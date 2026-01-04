package api

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"fmt"
	"io"
	"time"

	"github.com/google/uuid"
	"github.com/trebent/kerberos/internal/db"
	"github.com/trebent/zerologr"
)

//go:generate go tool oapi-codegen -config config.yaml -o ./gen.go oas.yaml

type impl struct {
	db db.SQLClient
}

var (
	_ StrictServerInterface = (*impl)(nil)

	queryCreateOrg = "INSERT INTO organisations (name) VALUES(@name);"
	// queryCreateGroup = "INSERT INTO groups (name, organisation) VALUES(@name, @orgID);".
	queryCreateUser = "INSERT INTO users (name, salt, hashed_password, organisation, administrator) VALUES(@name, @salt, @hashed_password, @orgID, @isAdmin);"

	queryLoginLookup        = "SELECT id, name, salt, hashed_password, organisation FROM users WHERE name = @username;"
	queryCreateSession      = "INSERT INTO sessions (user_id, session_id, expires) VALUES(@userID, @session, @expires);"
	quertGetSession         = "SELECT user_id, session_id, expires FROM sessions WHERE session_id = @sessionID;"
	queryDeleteUserSessions = "DELETE FROM sessions WHERE user_id = @userID;"
)

const sessionExpiry = 15 * time.Minute

func NewSSI(db db.SQLClient) StrictServerInterface {
	return &impl{db}
}

// Login implements [StrictServerInterface].
func (i *impl) Login(ctx context.Context, req LoginRequestObject) (LoginResponseObject, error) {
	rows, err := i.db.Query(
		ctx,
		queryLoginLookup,
		sql.NamedArg{Name: "username", Value: req.Body.Username},
	)
	if err != nil {
		zerologr.Error(err, "Failed to query for a user")
		return Login500JSONResponse{Message: "Internal error."}, nil
	}

	if !rows.Next() {
		if err := rows.Err(); err != nil {
			zerologr.Error(err, "Failed to prepare rows for scanning")
		}

		return Login401JSONResponse{Message: "Login failed."}, nil
	}

	id := 0
	name := ""
	salt := ""
	storedHashed := ""
	organisationID := 0
	err = rows.Scan(&id, &name, &salt, &storedHashed, &organisationID)
	//nolint:sqlclosecheck // won't help here
	_ = rows.Close()
	if err != nil {
		zerologr.Error(err, "Failed to scan for a matching user row")
		return Login500JSONResponse{Message: "Internal error."}, nil
	}

	decodedSalt, _ := hex.DecodeString(salt)
	hash := sha256.New()
	_, _ = hash.Write(decodedSalt)
	_, _ = hash.Write([]byte(req.Body.Password))
	inputHashed := hex.EncodeToString(hash.Sum(nil))

	if inputHashed != storedHashed {
		return Login401JSONResponse{Message: "Login failed."}, nil
	}
	zerologr.V(10).Info("User has logged in successfully", "username", req.Body.Username)

	sessionID := uuid.NewString()
	_, err = i.db.Exec(
		ctx,
		queryCreateSession,
		sql.NamedArg{Name: "userID", Value: id},
		sql.NamedArg{Name: "session", Value: sessionID},
		sql.NamedArg{Name: "expires", Value: time.Now().Add(sessionExpiry).UnixMilli()},
	)
	if err != nil {
		zerologr.Error(err, "Failed to store new session ID")
		return Login500JSONResponse{Message: "Internal error."}, nil
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
	rows, err := i.db.Query(
		ctx,
		quertGetSession,
		sql.NamedArg{Name: "sessionID", Value: req.Params.XKRBSession},
	)
	if err != nil {
		zerologr.Error(err, "Failed to query session")
		return Logout500JSONResponse{Message: "Internal error."}, nil
	}

	if !rows.Next() {
		if err := rows.Err(); err != nil {
			zerologr.Error(err, "Failed to prepare rows for scanning")
			return Logout500JSONResponse{Message: "Internal error."}, nil
		}

		zerologr.Info("Logout not possible since no session exists")
		return Logout204Response{}, nil
	}

	userID := 0
	err = rows.Scan(&userID, new(string), new(int64))
	//nolint:sqlclosecheck // won't help here
	_ = rows.Close()
	if err != nil {
		zerologr.Error(err, "Failed to scan row")
		return Logout500JSONResponse{Message: "Internal error."}, nil
	}
	zerologr.V(10).Info("Clearing user sessions", "user_id", userID)

	if _, err := i.db.Exec(
		ctx,
		queryDeleteUserSessions,
		sql.NamedArg{Name: "userID", Value: userID},
	); err != nil {
		zerologr.Error(err, "Failed to clear user sessions")
		return Logout500JSONResponse{Message: "Internal error."}, nil
	}

	return Logout204Response{}, nil
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
	ctx context.Context,
	req CreateOrganisationRequestObject,
) (CreateOrganisationResponseObject, error) {
	tx, err := i.db.Begin(ctx)
	if err != nil {
		zerologr.Error(err, "Failed to start transaction")
		return CreateOrganisation500JSONResponse{
			Message: "Internal error.",
		}, nil
	}
	//nolint:errcheck // no reason to
	defer tx.Rollback() // Just in case

	// Create an organisation.
	res, err := tx.Exec(ctx, queryCreateOrg, sql.NamedArg{Name: "name", Value: req.Body.Name})
	if err != nil {
		zerologr.Error(err, "Failed to create org")
		return CreateOrganisation500JSONResponse{
			Message: "Internal error.",
		}, nil
	}
	id, _ := res.LastInsertId()
	zerologr.Info(fmt.Sprintf("Created organisation with id %d", id))

	// Create an admin user for the organisation.
	adminUsername := fmt.Sprintf("%s-%s", "admin", req.Body.Name)

	adminPassword, salt, hashedAdminPassword := makePassword("")
	_, err = tx.Exec(
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
		return CreateOrganisation500JSONResponse{
			Message: "Internal error.",
		}, nil
	}

	err = tx.Commit()
	if err != nil {
		zerologr.Error(err, "Failed to commit transaction")
		return CreateOrganisation500JSONResponse{
			Message: "Internal error.",
		}, nil
	}

	return CreateOrganisation200JSONResponse{
		Name:          req.Body.Name,
		AdminPassword: adminPassword,
		AdminUsername: adminUsername,
	}, nil
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

// makePassword creates a random password if the input password is "". It will return the input/generated
// password, the salt, and the hashed version of the password.
func makePassword(password string) (string, string, string) {
	if password == "" {
		password = uuid.NewString()
	}
	hash := sha256.New()

	const saltBytes = 32
	salt := make([]byte, saltBytes)
	_, err := io.ReadFull(rand.Reader, salt)
	if err != nil {
		panic(err)
	}

	_, _ = hash.Write(salt)
	_, _ = hash.Write([]byte(password))
	return password, hex.EncodeToString(salt), hex.EncodeToString(hash.Sum(nil))
}
