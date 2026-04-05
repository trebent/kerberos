package admin

import (
	"bytes"
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/google/uuid"
	adminext "github.com/trebent/kerberos/internal/admin/extensions"
	"github.com/trebent/kerberos/internal/admin/model"
	"github.com/trebent/kerberos/internal/db"
	adminapi "github.com/trebent/kerberos/internal/oapi/admin"
	apierror "github.com/trebent/kerberos/internal/oapi/error"
	"github.com/trebent/kerberos/internal/util/password"
	"github.com/trebent/zerologr"
)

type (
	withExtensions interface {
		adminapi.StrictServerInterface

		// Extensions.

		// SetFlowFetcher sets the flow fetcher for the SSI, allowing it to serve flow metadata to the admin API.
		SetFlowFetcher(adminext.FlowFetcher)
		// SetOASBackend sets the OAS backend for the SSI, allowing it to serve OAS data to the admin API.
		SetOASBackend(adminext.OASBackend)
	}
	ssiOpts struct {
		SQLClient db.SQLClient

		ClientID     string
		ClientSecret string
	}
	impl struct {
		sqlClient db.SQLClient

		flowFetcher adminext.FlowFetcher
		oasBackend  adminext.OASBackend
	}
)

const (
	superSessionExpiry = 15 * time.Minute
)

var (
	_ withExtensions = (*impl)(nil)

	errNoSuperuser = errors.New("no superuser exists")
	errNoSession   = errors.New("no valid super session found")

	querySuperuser      = "SELECT id, name, salt, hashed_password FROM admin_users WHERE superuser = true;"
	querySession        = "SELECT s.user_id, s.session_id, u.superuser, s.expires FROM admin_sessions s JOIN admin_users u ON s.user_id = u.id WHERE s.session_id = @session_id;"
	insertSuperuser     = "INSERT INTO admin_users (name, salt, hashed_password, superuser) VALUES(@name, @salt, @hashed_password, true);"
	insertSession       = "INSERT INTO admin_sessions (session_id, user_id, expires) VALUES (@session_id, @user_id, @expires);"
	deleteSuperSessions = "DELETE FROM admin_sessions WHERE user_id = (SELECT id FROM admin_users WHERE superuser = true);"
)

func makeGenAPIError(msg string) adminapi.APIErrorResponse {
	return adminapi.APIErrorResponse{Errors: []string{msg}}
}

func newSSI(opts *ssiOpts) withExtensions {
	i := &impl{
		sqlClient: opts.SQLClient,

		oasBackend: &adminext.DummyOASBackend{},
	}
	i.bootstrapSuperuser(opts.ClientID, opts.ClientSecret)
	return i
}

func (i *impl) SetFlowFetcher(ff adminext.FlowFetcher) {
	i.flowFetcher = ff
}

func (i *impl) SetOASBackend(ob adminext.OASBackend) {
	i.oasBackend = ob
}

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

// GetFlow implements [adminapi.StrictServerInterface].
func (i *impl) GetFlow(
	_ context.Context,
	_ adminapi.GetFlowRequestObject,
) (adminapi.GetFlowResponseObject, error) {
	return adminapi.GetFlow200JSONResponse(i.flowFetcher.GetFlow()), nil
}

// GetBackendOAS implements [adminapi.StrictServerInterface].
func (i *impl) GetBackendOAS(
	_ context.Context,
	request adminapi.GetBackendOASRequestObject,
) (adminapi.GetBackendOASResponseObject, error) {
	oasData, err := i.oasBackend.GetOAS(request.Backend)
	if err != nil {
		return nil, err
	}

	return adminapi.GetBackendOAS200ApplicationyamlResponse{
		Body:          bytes.NewReader(oasData),
		ContentLength: int64(len(oasData)),
	}, nil
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

func (i *impl) querySuperuser() (*model.User, error) {
	rows, err := i.sqlClient.Query(context.Background(), querySuperuser)
	if err != nil {
		zerologr.Error(err, "Failed to query for superuser during session check")
		return nil, err
	}
	defer rows.Close()

	if rows.Next() {
		var user model.User
		if err := rows.Scan(
			&user.ID,
			&user.Username,
			&user.Salt,
			&user.HashedPassword,
		); err != nil {
			zerologr.Error(err, "Failed to scan superuser row")
			return nil, err
		}
		return &user, nil
	} else if err := rows.Err(); err != nil {
		zerologr.Error(err, "Error iterating superuser rows")
		return nil, err
	}

	return nil, errNoSuperuser
}

func (i *impl) querySession(sessionID string) (*model.Session, error) {
	rows, err := i.sqlClient.Query(
		context.Background(),
		querySession,
		sql.NamedArg{Name: "session_id", Value: sessionID},
	)
	if err != nil {
		zerologr.Error(err, "Failed to query for session")
		return nil, err
	}
	defer rows.Close()

	if rows.Next() {
		var session model.Session
		if err := rows.Scan(
			&session.UserID,
			&session.SessionID,
			&session.IsSuper,
			&session.Expires,
		); err != nil {
			zerologr.Error(err, "Failed to scan session row")
			return nil, err
		}
		return &session, nil
	} else if err := rows.Err(); err != nil {
		zerologr.Error(err, "Error iterating session rows")
		return nil, err
	}

	return nil, errNoSession
}
