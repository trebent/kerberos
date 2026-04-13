package admin

import (
	"context"
	"database/sql"

	"github.com/trebent/kerberos/internal/admin/model"
	"github.com/trebent/zerologr"
)

var (
	querySuperuser      = "SELECT id, name, salt, hashed_password FROM admin_users WHERE superuser = true;"
	querySession        = "SELECT s.user_id, s.session_id, u.superuser, s.expires FROM admin_sessions s JOIN admin_users u ON s.user_id = u.id WHERE s.session_id = @session_id;"
	insertSuperuser     = "INSERT INTO admin_users (name, salt, hashed_password, superuser) VALUES(@name, @salt, @hashed_password, true);"
	insertSession       = "INSERT INTO admin_sessions (session_id, user_id, expires) VALUES (@session_id, @user_id, @expires);"
	deleteSuperSessions = "DELETE FROM admin_sessions WHERE user_id = (SELECT id FROM admin_users WHERE superuser = true);"
)

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
