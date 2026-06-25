package admin

import (
	"context"
	"database/sql"
	"errors"
	"slices"
	"time"

	"github.com/trebent/kerberos/internal/admin/model"
	"github.com/trebent/kerberos/internal/db"
	"github.com/trebent/kerberos/internal/db/postgres"
	adminapi "github.com/trebent/kerberos/internal/oapi/admin"
	"github.com/trebent/kerberos/internal/util/password"
	"github.com/trebent/zerologr"
)

// --- SQL queries ---

const (
	// Superuser / sessions (existing).
	selectSuperuser     = "SELECT id, name, salt, hashed_password FROM admin_users WHERE superuser = true;"
	selectAdminSession  = "SELECT s.user_id, s.session_id, u.superuser, s.expires FROM admin_sessions s JOIN admin_users u ON s.user_id = u.id WHERE s.session_id = @session_id;"
	insertSuperuser     = "INSERT INTO admin_users (name, salt, hashed_password, superuser) VALUES(@name, @salt, @hashed_password, true);"
	insertSession       = "INSERT INTO admin_sessions (session_id, user_id, expires) VALUES (@session_id, @user_id, @expires);"
	deleteSuperSessions = "DELETE FROM admin_sessions WHERE user_id = (SELECT id FROM admin_users WHERE superuser = true);"

	// Users.
	selectAdminUsers         = "SELECT id, name FROM admin_users WHERE superuser = false;"
	selectAdminUser          = "SELECT id, name FROM admin_users WHERE id = @userID AND superuser = false;"
	selectAdminLoginUser     = "SELECT id, salt, hashed_password FROM admin_users WHERE name = @username AND superuser = false;"
	selectAdminUserAuth      = "SELECT salt, hashed_password FROM admin_users WHERE id = @userID AND superuser = false;"
	insertAdminUser          = "INSERT INTO admin_users (name, salt, hashed_password, superuser) VALUES(@name, @salt, @hashedPassword, false);"
	insertAdminUserReturning = "INSERT INTO admin_users (name, salt, hashed_password, superuser) VALUES(@name, @salt, @hashedPassword, false) RETURNING id"
	updateAdminUser          = "UPDATE admin_users SET name = @name WHERE id = @userID AND superuser = false;"
	deleteAdminUser          = "DELETE FROM admin_users WHERE id = @userID AND superuser = false;"
	//nolint:gosec // not a password
	updateAdminUserPassword = "UPDATE admin_users SET salt = @salt, hashed_password = @hashedPassword WHERE id = @userID AND superuser = false;"

	// Groups.
	selectAdminGroups         = "SELECT id, name FROM admin_groups;"
	selectAdminGroup          = "SELECT id, name FROM admin_groups WHERE id = @groupID;"
	insertAdminGroup          = "INSERT INTO admin_groups (name) VALUES(@name);"
	insertAdminGroupReturning = "INSERT INTO admin_groups (name) VALUES(@name) RETURNING id"
	updateAdminGroup          = "UPDATE admin_groups SET name = @name WHERE id = @groupID;"
	deleteAdminGroup          = "DELETE FROM admin_groups WHERE id = @groupID;"

	// Permissions.
	selectAdminPermissions = "SELECT id, name FROM admin_permissions;"
	insertAdminPermission  = "INSERT INTO admin_permissions (id, name) VALUES(@id, @name);"

	// Group permission bindings.
	selectGroupPermissions       = "SELECT p.id, p.name FROM admin_group_permission_bindings gpb INNER JOIN admin_permissions p ON gpb.permission_id = p.id WHERE gpb.group_id = @groupID;"
	deleteAdminGroupPermBindings = "DELETE FROM admin_group_permission_bindings WHERE group_id = @groupID;"
	insertAdminGroupPermBinding  = "INSERT INTO admin_group_permission_bindings (group_id, permission_id) VALUES (@groupID, @permissionID);"
	selectUserPermissionIDs      = "SELECT DISTINCT gpb.permission_id FROM admin_group_bindings gb INNER JOIN admin_group_permission_bindings gpb ON gb.group_id = gpb.group_id WHERE gb.user_id = @userID;"

	// Group bindings.
	selectAdminUserGroups   = "SELECT gb.group_id, g.name FROM admin_group_bindings gb INNER JOIN admin_groups g ON gb.group_id = g.id WHERE gb.user_id = @userID;"
	deleteAdminGroupBinding = "DELETE FROM admin_group_bindings WHERE user_id = @userID AND group_id = @groupID;"
	insertAdminGroupBinding = "INSERT INTO admin_group_bindings (user_id, group_id) VALUES (@userID, @groupID);"

	// Sessions.
	deleteAdminSession = "DELETE FROM admin_sessions WHERE session_id = @sessionID;"

	// Named arg keys.
	argBackend        = "backend"
	argExpiresAt      = "expires_at"
	argUserID         = "userID"
	argName           = "name"
	argSalt           = "salt"
	argHashedPassword = "hashedPassword"
	argGroupID        = "groupID"

	// Debug sessions.
	insertDebugSession          = "INSERT INTO admin_debug_sessions (backend, expires_at) VALUES(@backend, @expires_at);"
	insertDebugSessionReturning = "INSERT INTO admin_debug_sessions (backend, expires_at) VALUES(@backend, @expires_at) RETURNING id"
	selectDebugSessions         = "SELECT id, backend, started_at, expires_at, stopped_at FROM admin_debug_sessions WHERE backend = @backend;"
	selectDebugSession          = "SELECT id, backend, started_at, expires_at, stopped_at FROM admin_debug_sessions WHERE backend = @backend AND id = @id;"
	updateDebugSession          = "UPDATE admin_debug_sessions SET stopped_at = @stopped_at, expires_at = @expires_at WHERE backend = @backend AND id = @id;"
	deleteDebugSession          = "DELETE FROM admin_debug_sessions WHERE backend = @backend AND id = @id;"

	insertDebugSessionCall          = "INSERT INTO admin_debug_session_calls (session_id, started_at, stopped_at, url, method, status_code) VALUES(@session_id, @started_at, @stopped_at, @url, @method, @status_code);"
	insertDebugSessionCallReturning = "INSERT INTO admin_debug_session_calls (session_id, started_at, stopped_at, url, method, status_code) VALUES(@session_id, @started_at, @stopped_at, @url, @method, @status_code) RETURNING id"
	selectDebugSessionCalls         = "SELECT id, started_at, stopped_at, url, method, status_code FROM admin_debug_session_calls WHERE session_id = @session_id ORDER BY stopped_at DESC;"
	selectDebugSessionCall          = "SELECT id, started_at, stopped_at, url, method, status_code FROM admin_debug_session_calls WHERE id = @id ORDER BY stopped_at DESC;"

	insertDebugSessionFlowTransition  = "INSERT INTO admin_debug_session_call_flow_transitions (call_id, component, direction, started_at, stopped_at, result, failure_cause) VALUES(@call_id, @component, @direction, @started_at, @stopped_at, @result, @failure_cause);"
	selectDebugSessionFlowTransitions = "SELECT component, direction, started_at, stopped_at, result, failure_cause FROM admin_debug_session_call_flow_transitions WHERE call_id = @call_id ORDER BY started_at ASC;"

	sessionExpiry = 15 * time.Minute
)

var errRowNotFound = errors.New("row not found")

func dbListDebugSessions(
	ctx context.Context,
	client db.SQLClient,
	backend string,
) ([]adminapi.DebugSession, error) {
	rows, err := client.Query(
		ctx,
		selectDebugSessions,
		sql.NamedArg{Name: argBackend, Value: backend},
	)
	if err != nil {
		zerologr.Error(err, "Failed to query debug sessions")
		return nil, err
	}
	defer rows.Close()

	sessions := make([]adminapi.DebugSession, 0)
	for rows.Next() {
		var (
			session   adminapi.DebugSession
			startedAt db.TimeString
			expiresAt db.TimeString
		)
		if err := rows.Scan(
			&session.Id,
			&session.Backend,
			&startedAt,
			&expiresAt,
			db.NullTimeScanner{T: &session.StoppedAt},
		); err != nil {
			zerologr.Error(err, "Failed to scan debug session row")
			return nil, err
		}
		session.StartedAt = startedAt.Time
		session.ExpiresAt = expiresAt.Time
		sessions = append(sessions, session)
	}
	if err := rows.Err(); err != nil {
		zerologr.Error(err, "Failed to iterate debug session rows")
		return nil, err
	}

	return sessions, nil
}

func dbCreateDebugSession(
	ctx context.Context,
	client db.SQLClient,
	backend string,
	expiresAt time.Time,
) (int64, error) {
	if client.Dialect() == db.PostgresDialect {
		return postgres.InsertReturningID(ctx, client, insertDebugSessionReturning,
			sql.NamedArg{Name: argBackend, Value: backend},
			sql.NamedArg{Name: argExpiresAt, Value: expiresAt},
		)
	}

	res, err := client.Exec(
		ctx,
		insertDebugSession,
		sql.NamedArg{Name: argBackend, Value: backend},
		sql.NamedArg{Name: argExpiresAt, Value: expiresAt.UTC().Format(time.RFC3339Nano)},
	)
	if err != nil {
		zerologr.Error(err, "Failed to insert debug session")
		return 0, err
	}

	id, err := res.LastInsertId()
	if err != nil {
		zerologr.Error(err, "Failed to get last insert ID for debug session")
		return 0, err
	}
	return id, nil
}

func dbGetDebugSession(
	ctx context.Context,
	client db.SQLClient,
	backend string,
	id int64,
) (*adminapi.DebugSession, error) {
	rows, err := client.Query(
		ctx,
		selectDebugSession,
		sql.NamedArg{Name: argBackend, Value: backend},
		sql.NamedArg{Name: "id", Value: id},
	)
	if err != nil {
		zerologr.Error(err, "Failed to get debug session")
		return nil, err
	}
	defer rows.Close()

	if rows.Next() {
		var (
			session   = &adminapi.DebugSession{}
			startedAt db.TimeString
			expiresAt db.TimeString
		)
		if err := rows.Scan(
			&session.Id,
			&session.Backend,
			&startedAt,
			&expiresAt,
			db.NullTimeScanner{T: &session.StoppedAt},
		); err != nil {
			zerologr.Error(err, "Failed to scan debug session row")
			return nil, err
		}
		session.StartedAt = startedAt.Time
		session.ExpiresAt = expiresAt.Time
		return session, nil
	} else if err := rows.Err(); err != nil {
		zerologr.Error(err, "Error iterating debug session rows")
		return nil, err
	}

	return nil, errRowNotFound
}

func dbUpdateDebugSession(
	ctx context.Context,
	client db.SQLClient,
	debugSession adminapi.DebugSession,
) error {
	expiresAt := any(debugSession.ExpiresAt)
	stoppedAt := any(debugSession.StoppedAt)

	// Override if SQLite.
	if client.Dialect() == db.SQLiteDialect {
		expiresAt = debugSession.ExpiresAt.UTC().Format(time.RFC3339Nano)
		if debugSession.StoppedAt != nil {
			stoppedAt = debugSession.StoppedAt.UTC().Format(time.RFC3339Nano)
		}
	}

	_, err := client.Exec(
		ctx,
		updateDebugSession,
		sql.NamedArg{Name: "stopped_at", Value: stoppedAt},
		sql.NamedArg{Name: argExpiresAt, Value: expiresAt},
		sql.NamedArg{Name: argBackend, Value: debugSession.Backend},
		sql.NamedArg{Name: "id", Value: debugSession.Id},
	)
	if err != nil {
		zerologr.Error(err, "Failed to update debug session")
	}
	return err
}

func dbDeleteDebugSession(
	ctx context.Context,
	client db.SQLClient,
	backend string,
	id int64,
) error {
	_, err := client.Exec(
		ctx,
		deleteDebugSession,
		sql.Named(argBackend, backend),
		sql.Named("id", id),
	)
	if err != nil {
		zerologr.Error(err, "Failed to delete debug session")
		return err
	}

	return nil
}

func dbCreateDebugSessionCall(
	ctx context.Context,
	client db.SQLClient,
	sessionID int64,
	call adminapi.DebugSessionCall,
) (int64, error) {
	startedAt := any(call.StartedAt)
	stoppedAt := any(call.StoppedAt)

	if client.Dialect() == db.SQLiteDialect {
		startedAt = call.StartedAt.UTC().Format(time.RFC3339Nano)
		stoppedAt = call.StoppedAt.UTC().Format(time.RFC3339Nano)
	}

	// Insert call and record the ID.
	var (
		callID int64
		err    error
	)
	if client.Dialect() == db.PostgresDialect {
		callID, err = postgres.InsertReturningID(ctx, client, insertDebugSessionCallReturning,
			sql.Named("session_id", sessionID),
			sql.Named("started_at", startedAt),
			sql.Named("stopped_at", stoppedAt),
			sql.Named("url", call.Url),
			sql.Named("method", call.Method),
			sql.Named("status_code", call.StatusCode),
		)
		if err != nil {
			return 0, err
		}
	} else {
		res, err := client.Exec(
			ctx,
			insertDebugSessionCall,
			sql.Named("session_id", sessionID),
			sql.Named("started_at", startedAt),
			sql.Named("stopped_at", stoppedAt),
			sql.Named("url", call.Url),
			sql.Named("method", call.Method),
			sql.Named("status_code", call.StatusCode),
		)
		if err != nil {
			zerologr.Error(err, "Failed to insert debug session call")
			return 0, err
		}

		callID, err = res.LastInsertId()
		if err != nil {
			zerologr.Error(err, "Failed to get last insert ID for debug session")
			return 0, err
		}
	}

	// Insert transitions, link them to the recorded call ID.
	for _, transition := range call.FlowTransitions {
		startedAt := any(transition.StartedAt)
		stoppedAt := any(transition.StoppedAt)

		if client.Dialect() == db.SQLiteDialect {
			startedAt = transition.StartedAt.UTC().Format(time.RFC3339Nano)
			stoppedAt = transition.StoppedAt.UTC().Format(time.RFC3339Nano)
		}

		if _, err := client.Exec(
			ctx,
			insertDebugSessionFlowTransition,
			sql.Named("call_id", callID),
			sql.Named("component", transition.Component),
			sql.Named("started_at", startedAt),
			sql.Named("stopped_at", stoppedAt),
			sql.Named("direction", transition.Direction),
			sql.Named("result", transition.Result.Outcome),
			sql.Named("failure_cause", transition.Result.Cause),
		); err != nil {
			zerologr.Error(err, "Failed to insert debug session flow transition")
			return 0, err
		}
	}

	return callID, nil
}

func dbListDebugSessionCalls(
	ctx context.Context,
	client db.SQLClient,
	sessionID int64,
	includeTransitions bool,
) ([]adminapi.DebugSessionCall, error) {
	rows, err := client.Query(
		ctx,
		selectDebugSessionCalls,
		sql.Named("session_id", sessionID),
	)
	if err != nil {
		zerologr.Error(err, "Failed to query debug session calls")
		return nil, err
	}
	defer rows.Close()

	calls := make([]adminapi.DebugSessionCall, 0)
	for rows.Next() {
		var (
			call      adminapi.DebugSessionCall
			startedAt db.TimeString
			stoppedAt db.TimeString
		)
		if err := rows.Scan(
			&call.Id,
			&startedAt,
			&stoppedAt,
			&call.Url,
			&call.Method,
			&call.StatusCode,
		); err != nil {
			zerologr.Error(err, "Failed to scan debug session call row")
		}
		call.StartedAt = startedAt.Time
		call.StoppedAt = stoppedAt.Time
		calls = append(calls, call)
	}
	if err := rows.Err(); err != nil {
		zerologr.Error(err, "Failed to iterate admin user rows")
		return nil, err
	}

	if includeTransitions {
		// Fetch flow transitions per call.
		for i, call := range calls {
			transitions, err := dbListDebugSessionFlowTransitions(ctx, client, int64(call.Id))
			if err != nil {
				zerologr.Error(err, "Failed to list debug session flow transitions")
				return nil, err
			}
			calls[i].FlowTransitions = transitions
		}
	}

	return calls, nil
}

func dbGetDebugSessionCall(
	ctx context.Context,
	client db.SQLClient,
	callID int64,
) (*adminapi.DebugSessionCall, error) {
	rows, err := client.Query(
		ctx,
		selectDebugSessionCall,
		sql.Named("id", callID),
	)
	if err != nil {
		zerologr.Error(err, "Failed to query debug session call")
		return nil, err
	}
	defer rows.Close()

	if rows.Next() {
		var (
			call      = &adminapi.DebugSessionCall{}
			startedAt db.TimeString
			stoppedAt db.TimeString
		)
		if err := rows.Scan(
			&call.Id,
			&startedAt,
			&stoppedAt,
			&call.Url,
			&call.Method,
			&call.StatusCode,
		); err != nil {
			zerologr.Error(err, "Failed to scan debug session call row")
			return nil, err
		}
		call.StartedAt = startedAt.Time
		call.StoppedAt = stoppedAt.Time
		// Close cursor before issuing the nested flow-transitions query.
		// On SQLite (single connection) an open cursor blocks further queries.
		_ = rows.Close()

		transitions, err := dbListDebugSessionFlowTransitions(ctx, client, int64(call.Id))
		if err != nil {
			zerologr.Error(err, "Failed to list debug session flow transitions")
			return nil, err
		}
		call.FlowTransitions = transitions

		return call, nil
	} else if err := rows.Err(); err != nil {
		zerologr.Error(err, "Failed to iterate debug session call rows")
		return nil, err
	}

	return nil, errRowNotFound
}

func dbListDebugSessionFlowTransitions(
	ctx context.Context,
	client db.SQLClient,
	callID int64,
) ([]adminapi.FlowTransition, error) {
	rows, err := client.Query(
		ctx,
		selectDebugSessionFlowTransitions,
		sql.Named("call_id", callID),
	)
	if err != nil {
		zerologr.Error(err, "Failed to query debug session flow transitions")
		return nil, err
	}
	defer rows.Close()

	transitions := make([]adminapi.FlowTransition, 0)
	for rows.Next() {
		var (
			transition adminapi.FlowTransition
			startedAt  db.TimeString
			stoppedAt  db.TimeString
		)
		if err := rows.Scan(
			&transition.Component,
			&transition.Direction,
			&startedAt,
			&stoppedAt,
			&transition.Result.Outcome,
			&transition.Result.Cause,
		); err != nil {
			zerologr.Error(err, "Failed to scan debug session flow transition row")
			return nil, err
		}

		transition.StartedAt = startedAt.Time
		transition.StoppedAt = stoppedAt.Time
		transitions = append(transitions, transition)
	}

	if err := rows.Err(); err != nil {
		zerologr.Error(err, "Failed to iterate debug session flow transition rows")
		return nil, err
	}

	return transitions, nil
}

// --- Superuser / session helpers (used by ssi.go, middleware.go) ---

// dbGetSuperuser returns the superuser row.
// Returns (nil, errNoSuperuser) when no superuser exists.
func dbGetSuperuser(ctx context.Context, client db.SQLClient) (*model.User, error) {
	rows, err := client.Query(ctx, selectSuperuser)
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

	return nil, errRowNotFound
}

// dbGetSession returns a session by its session ID.
// Returns (nil, errNoSession) when no matching session exists.
func dbGetSession(
	ctx context.Context,
	client db.SQLClient,
	sessionID string,
) (*model.Session, error) {
	rows, err := client.Query(
		ctx,
		selectAdminSession,
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

	return nil, errRowNotFound
}

// --- Users ---

// dbGetUser returns a non-superuser admin user by ID.
// Returns (nil, errNoUser) when no matching user exists.
func dbGetUser(ctx context.Context, client db.SQLClient, userID int64) (*adminapi.User, error) {
	rows, err := client.Query(
		ctx,
		selectAdminUser,
		sql.NamedArg{Name: argUserID, Value: userID},
	)
	if err != nil {
		zerologr.Error(err, "Failed to query admin user")
		return nil, err
	}
	defer rows.Close()

	if !rows.Next() {
		if err := rows.Err(); err != nil {
			zerologr.Error(err, "Failed to iterate admin user rows")
			return nil, err
		}
		return nil, errRowNotFound
	}

	var u adminapi.User
	if err := rows.Scan(&u.Id, &u.Username); err != nil {
		zerologr.Error(err, "Failed to scan admin user row")
		return nil, err
	}

	return &u, nil
}

func dbListUsers(ctx context.Context, client db.SQLClient) ([]adminapi.User, error) {
	rows, err := client.Query(ctx, selectAdminUsers)
	if err != nil {
		zerologr.Error(err, "Failed to query admin users")
		return nil, err
	}
	defer rows.Close()

	users := make([]adminapi.User, 0)
	for rows.Next() {
		var u adminapi.User
		if err := rows.Scan(&u.Id, &u.Username); err != nil {
			zerologr.Error(err, "Failed to scan admin user row")
			return nil, err
		}
		users = append(users, u)
	}
	if err := rows.Err(); err != nil {
		zerologr.Error(err, "Failed to iterate admin user rows")
		return nil, err
	}

	return users, nil
}

func dbCreateUser(
	ctx context.Context,
	client db.SQLClient,
	username, salt, hashedPassword string,
) (int64, error) {
	if client.Dialect() == db.PostgresDialect {
		return postgres.InsertReturningID(ctx, client, insertAdminUserReturning,
			sql.NamedArg{Name: argName, Value: username},
			sql.NamedArg{Name: argSalt, Value: salt},
			sql.NamedArg{Name: argHashedPassword, Value: hashedPassword},
		)
	}
	res, err := client.Exec(
		ctx,
		insertAdminUser,
		sql.NamedArg{Name: argName, Value: username},
		sql.NamedArg{Name: argSalt, Value: salt},
		sql.NamedArg{Name: argHashedPassword, Value: hashedPassword},
	)
	if err != nil {
		zerologr.Error(err, "Failed to insert admin user")
		return 0, err
	}

	id, err := res.LastInsertId()
	if err != nil {
		zerologr.Error(err, "Failed to get last insert ID for admin user")
		return 0, err
	}
	return id, nil
}

func dbUpdateUser(ctx context.Context, client db.SQLClient, userID int64, username string) error {
	_, err := client.Exec(
		ctx,
		updateAdminUser,
		sql.NamedArg{Name: argName, Value: username},
		sql.NamedArg{Name: argUserID, Value: userID},
	)
	if err != nil {
		zerologr.Error(err, "Failed to update admin user")
	}
	return err
}

func dbDeleteUser(ctx context.Context, client db.SQLClient, userID int64) error {
	_, err := client.Exec(
		ctx,
		deleteAdminUser,
		sql.NamedArg{Name: argUserID, Value: userID},
	)
	if err != nil {
		zerologr.Error(err, "Failed to delete admin user")
	}
	return err
}

// dbGetUserAuth returns authentication credentials for a non-superuser admin user.
// Returns (nil, errNoUser) when no matching user exists.
func dbGetUserAuth(
	ctx context.Context,
	client db.SQLClient,
	userID int64,
) (*model.UserAuth, error) {
	rows, err := client.Query(
		ctx,
		selectAdminUserAuth,
		sql.NamedArg{Name: argUserID, Value: userID},
	)
	if err != nil {
		zerologr.Error(err, "Failed to query admin user auth")
		return nil, err
	}
	defer rows.Close()

	if !rows.Next() {
		if err := rows.Err(); err != nil {
			zerologr.Error(err, "Failed to iterate admin user auth rows")
			return nil, err
		}
		return nil, errRowNotFound
	}

	r := &model.UserAuth{}
	if err := rows.Scan(&r.Salt, &r.HashedPassword); err != nil {
		zerologr.Error(err, "Failed to scan admin user auth row")
		return nil, err
	}

	return r, nil
}

func dbUpdateUserPassword(
	ctx context.Context,
	client db.SQLClient,
	userID int64,
	salt, hashedPassword string,
) error {
	_, err := client.Exec(
		ctx,
		updateAdminUserPassword,
		sql.NamedArg{Name: argSalt, Value: salt},
		sql.NamedArg{Name: argHashedPassword, Value: hashedPassword},
		sql.NamedArg{Name: argUserID, Value: userID},
	)
	if err != nil {
		zerologr.Error(err, "Failed to update admin user password")
	}
	return err
}

// dbLoginLookup looks up a non-superuser admin user by username for authentication.
// Returns (nil, errNoUser) when no matching user exists.
func dbLoginLookup(
	ctx context.Context,
	client db.SQLClient,
	username string,
) (*model.SuperuserLoginUser, error) {
	rows, err := client.Query(
		ctx,
		selectAdminLoginUser,
		sql.NamedArg{Name: "username", Value: username},
	)
	if err != nil {
		zerologr.Error(err, "Failed to query admin user during login")
		return nil, err
	}
	defer rows.Close()

	if !rows.Next() {
		if err := rows.Err(); err != nil {
			zerologr.Error(err, "Failed to iterate admin login user rows")
			return nil, err
		}
		return nil, errRowNotFound
	}

	r := &model.SuperuserLoginUser{}
	if err := rows.Scan(&r.ID, &r.Salt, &r.HashedPassword); err != nil {
		zerologr.Error(err, "Failed to scan admin login user row")
		return nil, err
	}

	return r, nil
}

// --- Groups ---

// dbGetGroup returns an admin group by ID.
// Returns (nil, errNoGroup) when no matching group exists.
func dbGetGroup(ctx context.Context, client db.SQLClient, groupID int64) (*adminapi.Group, error) {
	rows, err := client.Query(
		ctx,
		selectAdminGroup,
		sql.NamedArg{Name: argGroupID, Value: groupID},
	)
	if err != nil {
		zerologr.Error(err, "Failed to query admin group")
		return nil, err
	}
	defer rows.Close()

	if !rows.Next() {
		if err := rows.Err(); err != nil {
			zerologr.Error(err, "Failed to iterate admin group rows")
			return nil, err
		}
		return nil, errRowNotFound
	}

	var g adminapi.Group
	if err := rows.Scan(&g.Id, &g.Name); err != nil {
		zerologr.Error(err, "Failed to scan admin group row")
		return nil, err
	}

	return &g, nil
}

func dbListGroups(ctx context.Context, client db.SQLClient) ([]adminapi.Group, error) {
	rows, err := client.Query(ctx, selectAdminGroups)
	if err != nil {
		zerologr.Error(err, "Failed to query admin groups")
		return nil, err
	}
	defer rows.Close()

	groups := make([]adminapi.Group, 0)
	for rows.Next() {
		var g adminapi.Group
		if err := rows.Scan(&g.Id, &g.Name); err != nil {
			zerologr.Error(err, "Failed to scan admin group row")
			return nil, err
		}
		groups = append(groups, g)
	}
	if err := rows.Err(); err != nil {
		zerologr.Error(err, "Failed to iterate admin group rows")
		return nil, err
	}

	return groups, nil
}

func dbCreateGroup(ctx context.Context, client db.SQLClient, name string) (int64, error) {
	if client.Dialect() == db.PostgresDialect {
		return postgres.InsertReturningID(ctx, client, insertAdminGroupReturning,
			sql.NamedArg{Name: argName, Value: name},
		)
	}
	res, err := client.Exec(
		ctx,
		insertAdminGroup,
		sql.NamedArg{Name: argName, Value: name},
	)
	if err != nil {
		zerologr.Error(err, "Failed to insert admin group")
		return 0, err
	}

	id, err := res.LastInsertId()
	if err != nil {
		zerologr.Error(err, "Failed to get last insert ID for admin group")
		return 0, err
	}
	return id, nil
}

func dbUpdateGroup(ctx context.Context, client db.SQLClient, groupID int64, name string) error {
	_, err := client.Exec(
		ctx,
		updateAdminGroup,
		sql.NamedArg{Name: argName, Value: name},
		sql.NamedArg{Name: argGroupID, Value: groupID},
	)
	if err != nil {
		zerologr.Error(err, "Failed to update admin group")
	}
	return err
}

func dbDeleteGroup(ctx context.Context, client db.SQLClient, groupID int64) error {
	_, err := client.Exec(
		ctx,
		deleteAdminGroup,
		sql.NamedArg{Name: argGroupID, Value: groupID},
	)
	if err != nil {
		zerologr.Error(err, "Failed to delete admin group")
	}
	return err
}

// --- Group bindings ---

func dbListGroupBindings(
	ctx context.Context,
	client db.SQLClient,
	userID int64,
) ([]*model.GroupBinding, error) {
	rows, err := client.Query(
		ctx,
		selectAdminUserGroups,
		sql.NamedArg{Name: argUserID, Value: userID},
	)
	if err != nil {
		zerologr.Error(err, "Failed to query admin user group bindings")
		return nil, err
	}
	defer rows.Close()

	bindings := make([]*model.GroupBinding, 0)
	for rows.Next() {
		b := &model.GroupBinding{}
		if err := rows.Scan(&b.GroupID, &b.Name); err != nil {
			zerologr.Error(err, "Failed to scan admin group binding row")
			return nil, err
		}
		bindings = append(bindings, b)
	}
	if err := rows.Err(); err != nil {
		zerologr.Error(err, "Failed to iterate admin group binding rows")
		return nil, err
	}

	return bindings, nil
}

// dbUpdateUserGroupBindings atomically updates a user's group memberships to exactly match desiredGroupIDs.
func dbUpdateUserGroupBindings(
	ctx context.Context,
	client db.SQLClient,
	userID int64,
	desiredGroupIDs []int,
) error {
	bindings, err := dbListGroupBindings(ctx, client, userID)
	if err != nil {
		return err
	}

	toDelete := make([]*model.GroupBinding, 0)
	for _, b := range bindings {
		if !slices.Contains(desiredGroupIDs, int(b.GroupID)) {
			toDelete = append(toDelete, b)
		}
	}

	tx, err := client.Begin(ctx)
	if err != nil {
		zerologr.Error(err, "Failed to start transaction")
		return err
	}
	//nolint:errcheck // intentional: no-op if already committed
	defer tx.Rollback()

	for _, b := range toDelete {
		if _, err := tx.Exec(
			ctx,
			deleteAdminGroupBinding,
			sql.NamedArg{Name: argUserID, Value: userID},
			sql.NamedArg{Name: argGroupID, Value: b.GroupID},
		); err != nil {
			zerologr.Error(err, "Failed to delete admin group binding")
			return err
		}
		bindings = slices.DeleteFunc(
			bindings,
			func(gb *model.GroupBinding) bool { return gb.GroupID == b.GroupID },
		)
	}

	for _, groupID := range desiredGroupIDs {
		if !slices.ContainsFunc(
			bindings,
			func(b *model.GroupBinding) bool { return int(b.GroupID) == groupID },
		) {
			if _, err := tx.Exec(
				ctx,
				insertAdminGroupBinding,
				sql.NamedArg{Name: argUserID, Value: userID},
				sql.NamedArg{Name: argGroupID, Value: groupID},
			); err != nil {
				zerologr.Error(err, "Failed to insert admin group binding")
				return err
			}
		}
	}

	if err := tx.Commit(); err != nil {
		zerologr.Error(err, "Failed to commit admin group binding transaction")
		return err
	}

	return nil
}

// --- Sessions ---

func dbCreateSession(
	ctx context.Context,
	client db.SQLClient,
	userID int64,
	sessionID string,
) error {
	_, err := client.Exec(
		ctx,
		insertSession,
		sql.NamedArg{Name: "user_id", Value: userID},
		sql.NamedArg{Name: "session_id", Value: sessionID},
		sql.NamedArg{Name: "expires", Value: time.Now().Add(sessionExpiry).UnixMilli()},
	)
	if err != nil {
		zerologr.Error(err, "Failed to store admin session")
	}
	return err
}

func dbDeleteSession(ctx context.Context, client db.SQLClient, sessionID string) error {
	_, err := client.Exec(
		ctx,
		deleteAdminSession,
		sql.NamedArg{Name: "sessionID", Value: sessionID},
	)
	if err != nil {
		zerologr.Error(err, "Failed to delete admin session")
	}
	return err
}

// bootstrapSuperuser checks if a super user exists and if not, creates one with the provided credentials.
// This is to allow bootstrapping of the first super user. Subsequent calls to this function will not have any effect.
// This is to prevent re-provisioning of the super-user, potentially allowing an attacker to reset powerful credentials.
func dbBootstrapSuperuser(client db.SQLClient, clientID, clientSecret string) error {
	// check if a super user already exists.
	rows, err := client.Query(context.Background(), selectSuperuser)
	if err != nil {
		return err
	}
	defer rows.Close()

	// No row found, check if due to an error.
	if !rows.Next() {
		if err := rows.Err(); err != nil {
			return err
		}

		// No super user exists, create one with the provided credentials.
		_, salt, hashedPassword := password.Make(clientSecret)
		if _, err := client.Exec(
			context.TODO(),
			insertSuperuser,
			sql.NamedArg{Name: argName, Value: clientID},
			sql.NamedArg{Name: argSalt, Value: salt},
			sql.NamedArg{Name: "hashed_password", Value: hashedPassword},
		); err != nil {
			return err
		}
	}

	return nil
}

// --- Permissions ---

// dbBootstrapPermissions inserts the fixed set of permissions if they do not yet exist.
func dbBootstrapPermissions(client db.SQLClient) error {
	perms := []struct {
		id   int64
		name string
	}{
		{PermissionIDFlowViewer, PermissionNameFlowViewer},
		{PermissionIDOASViewer, PermissionNameOASViewer},
		{PermissionIDBasicAuthOrgAdmin, PermissionNameBasicAuthOrgAdmin},
		{PermissionIDBasicAuthOrgViewer, PermissionNameBasicAuthOrgViewer},
		{PermissionIDAdminUserMgmtAdmin, PermissionNameAdminUserMgmtAdmin},
		{PermissionIDAdminUserMgmtViewer, PermissionNameAdminUserMgmtViewer},
		{PermissionIDDebugger, PermissionNameDebugger},
	}

	for _, p := range perms {
		if _, err := client.Exec(
			context.Background(),
			insertAdminPermission,
			sql.NamedArg{Name: "id", Value: p.id},
			sql.NamedArg{Name: argName, Value: p.name},
		); err != nil {
			if errors.Is(err, db.ErrUnique) {
				// Permission already exists, ignore error.
				zerologr.Info(
					"Permission already exists, skipping insert",
					"id",
					p.id,
					"name",
					p.name,
				)
				continue
			}

			zerologr.Error(err, "Failed to bootstrap permission", "name", p.name)
			return err
		}
	}

	return nil
}

// dbListPermissions returns all available permissions.
func dbListPermissions(ctx context.Context, client db.SQLClient) ([]adminapi.Permission, error) {
	rows, err := client.Query(ctx, selectAdminPermissions)
	if err != nil {
		zerologr.Error(err, "Failed to query admin permissions")
		return nil, err
	}
	defer rows.Close()

	perms := make([]adminapi.Permission, 0)
	for rows.Next() {
		var p adminapi.Permission
		if err := rows.Scan(&p.Id, &p.Name); err != nil {
			zerologr.Error(err, "Failed to scan admin permission row")
			return nil, err
		}
		perms = append(perms, p)
	}
	if err := rows.Err(); err != nil {
		zerologr.Error(err, "Failed to iterate admin permission rows")
		return nil, err
	}

	return perms, nil
}

// dbGetGroupPermissions returns the permissions assigned to the given group.
func dbGetGroupPermissions(
	ctx context.Context,
	client db.SQLClient,
	groupID int64,
) ([]adminapi.Permission, error) {
	rows, err := client.Query(
		ctx,
		selectGroupPermissions,
		sql.NamedArg{Name: argGroupID, Value: groupID},
	)
	if err != nil {
		zerologr.Error(err, "Failed to query group permissions")
		return nil, err
	}
	defer rows.Close()

	perms := make([]adminapi.Permission, 0)
	for rows.Next() {
		var p adminapi.Permission
		if err := rows.Scan(&p.Id, &p.Name); err != nil {
			zerologr.Error(err, "Failed to scan group permission row")
			return nil, err
		}
		perms = append(perms, p)
	}
	if err := rows.Err(); err != nil {
		zerologr.Error(err, "Failed to iterate group permission rows")
		return nil, err
	}

	return perms, nil
}

// dbSetGroupPermissions atomically replaces a group's permission bindings with the provided set.
func dbSetGroupPermissions(
	ctx context.Context,
	client db.SQLClient,
	groupID int64,
	permissionIDs []int,
) error {
	tx, err := client.Begin(ctx)
	if err != nil {
		zerologr.Error(err, "Failed to start transaction for group permission bindings")
		return err
	}
	//nolint:errcheck // Rollback is a best-effort cleanup. If Commit already succeeded, Rollback is a no-op.
	// Errors from Rollback are intentionally ignored in all cases as they cannot be recovered from here.
	defer tx.Rollback()

	if _, err := tx.Exec(
		ctx,
		deleteAdminGroupPermBindings,
		sql.NamedArg{Name: argGroupID, Value: groupID},
	); err != nil {
		zerologr.Error(err, "Failed to delete group permission bindings")
		return err
	}

	for _, permID := range permissionIDs {
		if _, err := tx.Exec(
			ctx,
			insertAdminGroupPermBinding,
			sql.NamedArg{Name: argGroupID, Value: groupID},
			sql.NamedArg{Name: "permissionID", Value: permID},
		); err != nil {
			zerologr.Error(err, "Failed to insert group permission binding")
			return err
		}
	}

	if err := tx.Commit(); err != nil {
		zerologr.Error(err, "Failed to commit group permission bindings transaction")
		return err
	}

	return nil
}

// dbGetUserPermissionIDs returns all permission IDs available to the given user via their group memberships.
func dbGetUserPermissionIDs(
	ctx context.Context,
	client db.SQLClient,
	userID int64,
) ([]int64, error) {
	rows, err := client.Query(
		ctx,
		selectUserPermissionIDs,
		sql.NamedArg{Name: argUserID, Value: userID},
	)
	if err != nil {
		zerologr.Error(err, "Failed to query user permission IDs")
		return nil, err
	}
	defer rows.Close()

	ids := make([]int64, 0)
	for rows.Next() {
		var id int64
		if err := rows.Scan(&id); err != nil {
			zerologr.Error(err, "Failed to scan user permission ID row")
			return nil, err
		}
		ids = append(ids, id)
	}
	if err := rows.Err(); err != nil {
		zerologr.Error(err, "Failed to iterate user permission ID rows")
		return nil, err
	}

	return ids, nil
}
