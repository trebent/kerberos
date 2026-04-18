package admin

import (
	"context"
	"database/sql"
	"errors"
	"slices"
	"time"

	"github.com/trebent/kerberos/internal/admin/model"
	"github.com/trebent/kerberos/internal/db"
	adminapi "github.com/trebent/kerberos/internal/oapi/admin"
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
	selectAdminUsers     = "SELECT id, name FROM admin_users WHERE superuser = false;"
	selectAdminUser      = "SELECT id, name FROM admin_users WHERE id = @userID AND superuser = false;"
	selectAdminLoginUser = "SELECT id, salt, hashed_password FROM admin_users WHERE name = @username AND superuser = false;"
	selectAdminUserAuth  = "SELECT salt, hashed_password FROM admin_users WHERE id = @userID AND superuser = false;"
	insertAdminUser      = "INSERT INTO admin_users (name, salt, hashed_password, superuser) VALUES(@name, @salt, @hashedPassword, false);"
	updateAdminUser      = "UPDATE admin_users SET name = @name WHERE id = @userID AND superuser = false;"
	deleteAdminUser      = "DELETE FROM admin_users WHERE id = @userID AND superuser = false;"
	//nolint:gosec // not a password
	updateAdminUserPassword = "UPDATE admin_users SET salt = @salt, hashed_password = @hashedPassword WHERE id = @userID AND superuser = false;"

	// Groups.
	selectAdminGroups = "SELECT id, name FROM admin_groups;"
	selectAdminGroup  = "SELECT id, name FROM admin_groups WHERE id = @groupID;"
	insertAdminGroup  = "INSERT INTO admin_groups (name) VALUES(@name);"
	updateAdminGroup  = "UPDATE admin_groups SET name = @name WHERE id = @groupID;"
	deleteAdminGroup  = "DELETE FROM admin_groups WHERE id = @groupID;"

	// Permissions.
	selectAdminPermissions = "SELECT id, name FROM admin_permissions;"
	insertAdminPermission  = "INSERT OR IGNORE INTO admin_permissions (id, name) VALUES(@id, @name);"

	// Group permission bindings.
	selectGroupPermissions          = "SELECT p.id, p.name FROM admin_group_permission_bindings gpb INNER JOIN admin_permissions p ON gpb.permission_id = p.id WHERE gpb.group_id = @groupID;"
	deleteAdminGroupPermBindings    = "DELETE FROM admin_group_permission_bindings WHERE group_id = @groupID;"
	insertAdminGroupPermBinding     = "INSERT INTO admin_group_permission_bindings (group_id, permission_id) VALUES (@groupID, @permissionID);"
	selectUserPermissionIDs         = "SELECT DISTINCT gpb.permission_id FROM admin_group_bindings gb INNER JOIN admin_group_permission_bindings gpb ON gb.group_id = gpb.group_id WHERE gb.user_id = @userID;"

	// Group bindings.
	selectAdminUserGroups   = "SELECT gb.group_id, g.name FROM admin_group_bindings gb INNER JOIN admin_groups g ON gb.group_id = g.id WHERE gb.user_id = @userID;"
	deleteAdminGroupBinding = "DELETE FROM admin_group_bindings WHERE user_id = @userID AND group_id = @groupID;"
	insertAdminGroupBinding = "INSERT INTO admin_group_bindings (user_id, group_id) VALUES (@userID, @groupID);"

	// Sessions.
	deleteAdminSession = "DELETE FROM admin_sessions WHERE session_id = @sessionID;"

	sessionExpiry = 15 * time.Minute
)

var (
	errNoUser  = errors.New("no user found")
	errNoGroup = errors.New("no group found")
)

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

	return nil, errNoSuperuser
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

	return nil, errNoSession
}

// --- Users ---

// dbGetUser returns a non-superuser admin user by ID.
// Returns (nil, errNoUser) when no matching user exists.
func dbGetUser(ctx context.Context, client db.SQLClient, userID int64) (*adminapi.User, error) {
	rows, err := client.Query(
		ctx,
		selectAdminUser,
		sql.NamedArg{Name: "userID", Value: userID},
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
		return nil, errNoUser
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
	res, err := client.Exec(
		ctx,
		insertAdminUser,
		sql.NamedArg{Name: "name", Value: username},
		sql.NamedArg{Name: "salt", Value: salt},
		sql.NamedArg{Name: "hashedPassword", Value: hashedPassword},
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
		sql.NamedArg{Name: "name", Value: username},
		sql.NamedArg{Name: "userID", Value: userID},
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
		sql.NamedArg{Name: "userID", Value: userID},
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
		sql.NamedArg{Name: "userID", Value: userID},
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
		return nil, errNoUser
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
		sql.NamedArg{Name: "salt", Value: salt},
		sql.NamedArg{Name: "hashedPassword", Value: hashedPassword},
		sql.NamedArg{Name: "userID", Value: userID},
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
		return nil, errNoUser
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
		sql.NamedArg{Name: "groupID", Value: groupID},
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
		return nil, errNoGroup
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
	res, err := client.Exec(
		ctx,
		insertAdminGroup,
		sql.NamedArg{Name: "name", Value: name},
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
		sql.NamedArg{Name: "name", Value: name},
		sql.NamedArg{Name: "groupID", Value: groupID},
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
		sql.NamedArg{Name: "groupID", Value: groupID},
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
		sql.NamedArg{Name: "userID", Value: userID},
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
			sql.NamedArg{Name: "userID", Value: userID},
			sql.NamedArg{Name: "groupID", Value: b.GroupID},
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
				sql.NamedArg{Name: "userID", Value: userID},
				sql.NamedArg{Name: "groupID", Value: groupID},
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

// --- Permissions ---

// dbBootstrapPermissions inserts the fixed set of permissions if they do not yet exist.
func dbBootstrapPermissions(ctx context.Context, client db.SQLClient) error {
	perms := []struct {
		id   int64
		name string
	}{
		{PermissionIDFlowViewer, PermissionNameFlowViewer},
		{PermissionIDOASViewer, PermissionNameOASViewer},
		{PermissionIDBasicAuthOrgAdmin, PermissionNameBasicAuthOrgAdmin},
		{PermissionIDBasicAuthOrgViewer, PermissionNameBasicAuthOrgViewer},
	}

	for _, p := range perms {
		if _, err := client.Exec(
			ctx,
			insertAdminPermission,
			sql.NamedArg{Name: "id", Value: p.id},
			sql.NamedArg{Name: "name", Value: p.name},
		); err != nil {
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
func dbGetGroupPermissions(ctx context.Context, client db.SQLClient, groupID int64) ([]adminapi.Permission, error) {
	rows, err := client.Query(
		ctx,
		selectGroupPermissions,
		sql.NamedArg{Name: "groupID", Value: groupID},
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
func dbSetGroupPermissions(ctx context.Context, client db.SQLClient, groupID int64, permissionIDs []int) error {
	tx, err := client.Begin(ctx)
	if err != nil {
		zerologr.Error(err, "Failed to start transaction for group permission bindings")
		return err
	}
	//nolint:errcheck // intentional: no-op if already committed
	defer tx.Rollback()

	if _, err := tx.Exec(
		ctx,
		deleteAdminGroupPermBindings,
		sql.NamedArg{Name: "groupID", Value: groupID},
	); err != nil {
		zerologr.Error(err, "Failed to delete group permission bindings")
		return err
	}

	for _, permID := range permissionIDs {
		if _, err := tx.Exec(
			ctx,
			insertAdminGroupPermBinding,
			sql.NamedArg{Name: "groupID", Value: groupID},
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
func dbGetUserPermissionIDs(ctx context.Context, client db.SQLClient, userID int64) ([]int64, error) {
	rows, err := client.Query(
		ctx,
		selectUserPermissionIDs,
		sql.NamedArg{Name: "userID", Value: userID},
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
