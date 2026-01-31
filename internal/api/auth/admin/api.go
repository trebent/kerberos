package adminapi

import (
	"context"
	"database/sql"
	"time"

	"github.com/google/uuid"
	apierror "github.com/trebent/kerberos/internal/api/error"
	"github.com/trebent/kerberos/internal/auth/model"
	"github.com/trebent/kerberos/internal/auth/util"
	"github.com/trebent/kerberos/internal/db"
	"github.com/trebent/zerologr"
)

//go:generate go tool oapi-codegen -config config.yaml -o ./gen.go ../../../openapi/administration.yaml

type (
	impl struct {
		db db.SQLClient

		// Administration information, populated when used.
		organisationID int64

		clientID     string
		clientSecret string
	}
)

var (
	queryCreateOrg = "INSERT INTO organisations (name) VALUES(@name);"
	queryGetOrg    = "SELECT id FROM organisations WHERE name = @orgName;"

	//nolint:gosec // false positive
	queryCreateSuperUser = "INSERT INTO users (name, salt, hashed_password, organisation_id, super_user) VALUES(@name, @salt, @hashed_password, @orgID, true);"

	queryLoginLookup = "SELECT id, name, salt, hashed_password, super_user, organisation_id FROM users WHERE organisation_id = @orgID AND name = @username;"

	queryCreateSession      = "INSERT INTO sessions (user_id, organisation_id, session_id, expires) VALUES(@userID, @orgID, @session, @expires);"
	queryDeleteUserSessions = "DELETE FROM sessions WHERE organisation_id = @orgID AND user_id = (SELECT id FROM users WHERE name = @username);"
)

const (
	adminOrgName = "administration"

	sessionExpiry = 15 * time.Minute
)

func makeGenAPIError(msg string) APIErrorResponse {
	return APIErrorResponse{Errors: []string{msg}}
}

func NewSSI(db db.SQLClient, clientID, clientSecret string) StrictServerInterface {
	return &impl{db: db, clientID: clientID, clientSecret: clientSecret}
}

// LoginSuperuser implements [StrictServerInterface].
func (i *impl) LoginSuperuser(
	ctx context.Context,
	request LoginSuperuserRequestObject,
) (LoginSuperuserResponseObject, error) {
	if request.Body.ClientId != i.clientID {
		return LoginSuperuser401JSONResponse(makeGenAPIError("Login failed.")), nil
	}

	org, err := i.getOrCreateOrg(ctx)
	if err != nil {
		return nil, err
	}
	i.organisationID = org.ID

	user, err := i.loginLookup(ctx)
	if err != nil {
		return nil, err
	}

	if !util.PasswordMatch(user.Salt, user.HashedPassword, request.Body.ClientSecret) {
		zerologr.Info("User login failed due to password mismatch")
		return LoginSuperuser401JSONResponse(makeGenAPIError("Login failed.")), nil
	}

	session, err := i.createSession(ctx, user)
	if err != nil {
		zerologr.Error(err, "Failed to store new session ID")
		return LoginSuperuser500JSONResponse(makeGenAPIError(apierror.ErrInternal.Error())), nil
	}

	return LoginSuperuser204Response{
		Headers: LoginSuperuser204ResponseHeaders{
			XKrbSession: session,
		},
	}, nil
}

// LogoutSuperuser implements [StrictServerInterface].
func (i *impl) LogoutSuperuser(
	ctx context.Context,
	_ LogoutSuperuserRequestObject,
) (LogoutSuperuserResponseObject, error) {
	org, err := i.getOrCreateOrg(ctx)
	if err != nil {
		//nolint:nilerr // that's the point
		return LogoutSuperuser500JSONResponse(makeGenAPIError(apierror.APIErrInternal.Error())), nil
	}
	i.organisationID = org.ID

	if _, err := i.db.Exec(
		ctx,
		queryDeleteUserSessions,
		sql.NamedArg{Name: "orgID", Value: i.organisationID},
		sql.NamedArg{Name: "username", Value: i.clientID},
	); err != nil {
		zerologr.Error(err, "Failed to clear user sessions")
		return LogoutSuperuser500JSONResponse(makeGenAPIError(apierror.APIErrInternal.Error())), nil
	}

	return LogoutSuperuser204Response{}, nil
}

func (i *impl) getOrCreateOrg(ctx context.Context) (*model.Organisation, error) {
	if i.organisationID != 0 {
		return &model.Organisation{ID: i.organisationID, Name: adminOrgName}, nil
	}

	rows, err := i.db.Query(
		ctx,
		queryGetOrg,
		sql.NamedArg{Name: "orgName", Value: adminOrgName},
	)
	if err != nil {
		zerologr.Error(err, "Failed to query administration organisation")
		return nil, apierror.APIErrInternal
	}

	scanned := rows.Next()
	if !scanned {
		if err := rows.Err(); err != nil {
			zerologr.Error(err, "Failed to prepare rows for scanning")
			return nil, apierror.APIErrInternal
		}
	}

	org := &model.Organisation{Name: adminOrgName}
	if scanned {
		err = rows.Scan(&org.ID)
		//nolint:sqlclosecheck // won't help here
		_ = rows.Close()
		if err != nil {
			zerologr.Error(err, "Failed to scan for a matching user row")
			return nil, apierror.APIErrInternal
		}
	} else {
		res, err := i.db.Exec(ctx, queryCreateOrg, sql.NamedArg{Name: "name", Value: org.Name})
		if err != nil {
			zerologr.Error(err, "Failed to create administration organisation")
			return nil, apierror.APIErrInternal
		}

		org.ID, _ = res.LastInsertId()
	}

	return org, nil
}

func (i *impl) loginLookup(ctx context.Context) (*model.User, error) {
	rows, err := i.db.Query(
		ctx,
		queryLoginLookup,
		sql.NamedArg{Name: "username", Value: i.clientID},
		sql.NamedArg{Name: "orgID", Value: i.organisationID},
	)
	if err != nil {
		zerologr.Error(err, "Failed to query user")
		return nil, apierror.APIErrInternal
	}

	scanned := rows.Next()
	if !scanned {
		if err := rows.Err(); err != nil {
			zerologr.Error(err, "Failed to prepare rows for scanning")
			return nil, apierror.APIErrInternal
		}
	}

	user := &model.User{}
	if scanned {
		err = rows.Scan(
			&user.ID,
			&user.Name,
			&user.Salt,
			&user.HashedPassword,
			&user.SuperUser,
			&user.OrganisationID,
		)
		//nolint:sqlclosecheck // won't help here
		_ = rows.Close()
		if err != nil {
			zerologr.Error(err, "Failed to scan for a matching user row")
			return nil, apierror.APIErrInternal
		}
	} else {
		zerologr.Info("Superuser did not exist, creating it now")
		_, salt, hashedPassword := util.MakePassword(i.clientSecret)
		res, err := i.db.Exec(
			ctx,
			queryCreateSuperUser,
			sql.NamedArg{Name: "name", Value: i.clientID},
			sql.NamedArg{Name: "salt", Value: salt},
			sql.NamedArg{Name: "hashed_password", Value: hashedPassword},
			sql.NamedArg{Name: "orgID", Value: i.organisationID},
		)
		if err != nil {
			zerologr.Error(err, "Failed to create super user")
			return nil, apierror.APIErrInternal
		}

		user.ID, _ = res.LastInsertId()
		user.OrganisationID = i.organisationID
		user.Name = i.clientID
		user.Salt = salt
		user.HashedPassword = hashedPassword
		user.SuperUser = true
	}

	return user, nil
}

func (i *impl) createSession(ctx context.Context, user *model.User) (string, error) {
	sessionID := uuid.NewString()
	_, err := i.db.Exec(
		ctx,
		queryCreateSession,
		sql.NamedArg{Name: "userID", Value: user.ID},
		sql.NamedArg{Name: "orgID", Value: user.OrganisationID},
		sql.NamedArg{Name: "session", Value: sessionID},
		sql.NamedArg{Name: "expires", Value: time.Now().Add(sessionExpiry).UnixMilli()},
	)

	return sessionID, err
}
