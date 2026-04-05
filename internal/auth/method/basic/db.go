package basic

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"slices"
	"time"

	models "github.com/trebent/kerberos/internal/auth/method/basic/model"
	"github.com/trebent/kerberos/internal/db"
	authbasicapi "github.com/trebent/kerberos/internal/oapi/auth/basic"
	"github.com/trebent/kerberos/internal/util/password"
	"github.com/trebent/zerologr"
)

const (
	// Organisations.
	insertOrg  = "INSERT INTO organisations (name) VALUES(@name);"
	deleteOrg  = "DELETE FROM organisations WHERE id = @orgID;"
	selectOrg  = "SELECT id, name FROM organisations WHERE id = @orgID;"
	selectOrgs = "SELECT id, name FROM organisations;"
	updateOrg  = "UPDATE organisations SET name = @name WHERE id = @orgID;"

	// Groups.
	insertGroup  = "INSERT INTO groups (name, organisation_id) VALUES(@name, @orgID);"
	deleteGroup  = "DELETE FROM groups WHERE organisation_id = @orgID AND id = @groupID;"
	selectGroup  = "SELECT id, name FROM groups WHERE id = @groupID AND organisation_id = @orgID;"
	selectGroups = "SELECT id, name FROM groups WHERE organisation_id = @orgID;"
	updateGroup  = "UPDATE groups SET name = @name WHERE id = @groupID AND organisation_id = @orgID;"

	// Users.
	insertUser     = "INSERT INTO users (name, salt, hashed_password, organisation_id, administrator) VALUES(@name, @salt, @hashedPassword, @orgID, @isAdmin);"
	deleteUser     = "DELETE FROM users WHERE id = @userID AND organisation_id = @orgID;"
	selectUser     = "SELECT id, name FROM users WHERE id = @userID AND organisation_id = @orgID;"
	selectUserAuth = "SELECT salt, hashed_password FROM users WHERE id = @userID AND organisation_id = @orgID;"
	selectUsers    = "SELECT id, name FROM users WHERE organisation_id = @orgID;"
	updateUser     = "UPDATE users SET name = @name WHERE id = @userID AND organisation_id = @orgID;"
	//nolint:gosec // not a password
	updateUserPassword = "UPDATE users SET salt = @salt, hashed_password = @hashedPassword WHERE id = @id;"
	selectLoginUser    = "SELECT id, name, salt, hashed_password, organisation_id FROM users WHERE organisation_id = @orgID AND name = @username;"

	// Group bindings.
	selectUserGroups    = "SELECT name FROM groups WHERE id IN (SELECT group_id FROM group_bindings WHERE user_id = @userID) AND organisation_id = @orgID;"
	selectGroupBindings = "SELECT g.id, g.name FROM group_bindings gb INNER JOIN groups g ON gb.group_id = g.id WHERE user_id = @userID AND organisation_id = @orgID;"
	deleteGroupBinding  = "DELETE FROM group_bindings WHERE user_id = @userID AND group_id = @groupID;"
	insertGroupBinding  = "INSERT INTO group_bindings (user_id, group_id) VALUES (@userID, (SELECT id FROM groups WHERE organisation_id = @orgID AND name = @groupName));"

	// Sessions.
	insertSession      = "INSERT INTO sessions (user_id, organisation_id, session_id, expires) VALUES(@userID, @orgID, @session, @expires);"
	selectSession      = "SELECT s.user_id, s.organisation_id, u.administrator, s.expires FROM sessions s INNER JOIN users u ON s.user_id = u.id WHERE session_id = @sessionID;"
	deleteUserSessions = "DELETE FROM sessions WHERE organisation_id = @orgID AND user_id = @userID;"

	sessionExpiry = 15 * time.Minute
)

var (
	errNoSession = errors.New("no valid session found")
	errNoUser    = errors.New("no user found")
	errNoGroup   = errors.New("no group found")
	errNoOrg     = errors.New("no organisation found")
)

// --- Package-level helpers (shared by impl and basic) ---

// dbGetSessionRow queries a session by ID and returns the scanned result.
// Returns (nil, errNoSession) when no matching session row exists.
func dbGetSessionRow(
	ctx context.Context,
	client db.SQLClient,
	sessionID string,
) (*models.Session, error) {
	rows, err := client.Query(ctx, selectSession, sql.NamedArg{Name: "sessionID", Value: sessionID})
	if err != nil {
		zerologr.Error(err, "Failed to query session")
		return nil, err
	}
	defer rows.Close()

	if !rows.Next() {
		if err := rows.Err(); err != nil {
			zerologr.Error(err, "Failed to iterate session rows")
			return nil, err
		}
		return nil, errNoSession
	}

	r := &models.Session{}
	if err := rows.Scan(&r.UserID, &r.OrgID, &r.Administrator, &r.Expires); err != nil {
		zerologr.Error(err, "Failed to scan session row")
		return nil, err
	}

	return r, nil
}

// dbGetUserGroupNames returns the names of all groups a user belongs to within an organisation.
func dbGetUserGroupNames(
	ctx context.Context,
	client db.SQLClient,
	orgID, userID int64,
) ([]string, error) {
	rows, err := client.Query(
		ctx,
		selectUserGroups,
		sql.NamedArg{Name: "orgID", Value: orgID},
		sql.NamedArg{Name: "userID", Value: userID},
	)
	if err != nil {
		zerologr.Error(err, "Failed to query user groups")
		return nil, err
	}
	defer rows.Close()

	groups := make([]string, 0)
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			zerologr.Error(err, "Failed to scan user group row")
			return nil, err
		}
		groups = append(groups, name)
	}
	if err := rows.Err(); err != nil {
		zerologr.Error(err, "Failed to iterate user group rows")
		return nil, err
	}

	return groups, nil
}

// --- impl methods: Sessions ---

func (i *impl) dbCreateSession(ctx context.Context, userID, orgID int64, sessionID string) error {
	_, err := i.db.Exec(
		ctx,
		insertSession,
		sql.NamedArg{Name: "userID", Value: userID},
		sql.NamedArg{Name: "orgID", Value: orgID},
		sql.NamedArg{Name: "session", Value: sessionID},
		sql.NamedArg{Name: "expires", Value: time.Now().Add(sessionExpiry).UnixMilli()},
	)
	if err != nil {
		zerologr.Error(err, "Failed to store new session")
	}
	return err
}

func (i *impl) dbDeleteUserSessions(ctx context.Context, orgID, userID int64) error {
	_, err := i.db.Exec(
		ctx,
		deleteUserSessions,
		sql.NamedArg{Name: "orgID", Value: orgID},
		sql.NamedArg{Name: "userID", Value: userID},
	)
	if err != nil {
		zerologr.Error(err, "Failed to delete user sessions")
	}
	return err
}

// --- impl methods: Users ---

// dbLoginLookup looks up a user by organisation and username for authentication.
// Returns (nil, errNoUser) when no matching user exists.
func (i *impl) dbLoginLookup(
	ctx context.Context,
	orgID int64,
	username string,
) (*models.LoginUser, error) {
	rows, err := i.db.Query(
		ctx,
		selectLoginUser,
		sql.NamedArg{Name: "orgID", Value: orgID},
		sql.NamedArg{Name: "username", Value: username},
	)
	if err != nil {
		zerologr.Error(err, "Failed to query user during login")
		return nil, err
	}
	defer rows.Close()

	if !rows.Next() {
		if err := rows.Err(); err != nil {
			zerologr.Error(err, "Failed to iterate login user rows")
			return nil, err
		}
		return nil, errNoUser
	}

	r := &models.LoginUser{}
	if err := rows.Scan(
		&r.ID,
		new(string),
		&r.Salt,
		&r.HashedPassword,
		&r.OrganisationID,
	); err != nil {
		zerologr.Error(err, "Failed to scan login user row")
		return nil, err
	}

	return r, nil
}

// dbGetUser returns a user by ID within an organisation.
// Returns (nil, errNoUser) when no matching user exists.
func (i *impl) dbGetUser(ctx context.Context, orgID, userID int64) (*authbasicapi.User, error) {
	rows, err := i.db.Query(
		ctx,
		selectUser,
		sql.NamedArg{Name: "orgID", Value: orgID},
		sql.NamedArg{Name: "userID", Value: userID},
	)
	if err != nil {
		zerologr.Error(err, "Failed to query user")
		return nil, err
	}
	defer rows.Close()

	if !rows.Next() {
		if err := rows.Err(); err != nil {
			zerologr.Error(err, "Failed to iterate user rows")
			return nil, err
		}
		return nil, errNoUser
	}

	var u authbasicapi.User
	if err := rows.Scan(&u.Id, &u.Name); err != nil {
		zerologr.Error(err, "Failed to scan user row")
		return nil, err
	}

	return &u, nil
}

// dbGetUserAuth returns the authentication credentials for a user by ID within an organisation.
// Returns (nil, errNoUser) when no matching user exists.
func (i *impl) dbGetUserAuth(ctx context.Context, orgID, userID int64) (*models.UserAuth, error) {
	rows, err := i.db.Query(
		ctx,
		selectUserAuth,
		sql.NamedArg{Name: "orgID", Value: orgID},
		sql.NamedArg{Name: "userID", Value: userID},
	)
	if err != nil {
		zerologr.Error(err, "Failed to query full user record")
		return nil, err
	}
	defer rows.Close()

	if !rows.Next() {
		if err := rows.Err(); err != nil {
			zerologr.Error(err, "Failed to iterate full user rows")
			return nil, err
		}
		return nil, errNoUser
	}

	r := &models.UserAuth{}
	if err = rows.Scan(&r.Salt, &r.HashedPassword); err != nil {
		zerologr.Error(err, "Failed to scan user auth row")
		return nil, err
	}

	return r, nil
}

func (i *impl) dbListUsers(ctx context.Context, orgID int64) ([]authbasicapi.User, error) {
	rows, err := i.db.Query(
		ctx,
		selectUsers,
		sql.NamedArg{Name: "orgID", Value: orgID},
	)
	if err != nil {
		zerologr.Error(err, "Failed to query users")
		return nil, err
	}
	defer rows.Close()

	users := make([]authbasicapi.User, 0)
	for rows.Next() {
		var u authbasicapi.User
		if err := rows.Scan(&u.Id, &u.Name); err != nil {
			zerologr.Error(err, "Failed to scan user row")
			return nil, err
		}
		users = append(users, u)
	}
	if err := rows.Err(); err != nil {
		zerologr.Error(err, "Failed to iterate user rows")
		return nil, err
	}

	return users, nil
}

func (i *impl) dbCreateUser(
	ctx context.Context,
	name, salt, hashedPassword string,
	orgID int64,
	isAdmin bool,
) (int64, error) {
	res, err := i.db.Exec(
		ctx,
		insertUser,
		sql.NamedArg{Name: "name", Value: name},
		sql.NamedArg{Name: "salt", Value: salt},
		sql.NamedArg{Name: "hashedPassword", Value: hashedPassword},
		sql.NamedArg{Name: "orgID", Value: orgID},
		sql.NamedArg{Name: "isAdmin", Value: isAdmin},
	)
	if err != nil {
		zerologr.Error(err, "Failed to insert user")
		return 0, err
	}

	id, _ := res.LastInsertId()
	return id, nil
}

func (i *impl) dbUpdateUser(ctx context.Context, orgID, userID int64, name string) error {
	_, err := i.db.Exec(
		ctx,
		updateUser,
		sql.NamedArg{Name: "name", Value: name},
		sql.NamedArg{Name: "orgID", Value: orgID},
		sql.NamedArg{Name: "userID", Value: userID},
	)
	if err != nil {
		zerologr.Error(err, "Failed to update user")
	}
	return err
}

func (i *impl) dbDeleteUser(ctx context.Context, orgID, userID int64) error {
	_, err := i.db.Exec(
		ctx,
		deleteUser,
		sql.NamedArg{Name: "orgID", Value: orgID},
		sql.NamedArg{Name: "userID", Value: userID},
	)
	if err != nil {
		zerologr.Error(err, "Failed to delete user")
	}
	return err
}

func (i *impl) dbUpdateUserPassword(
	ctx context.Context,
	userID int64,
	salt, hashedPassword string,
) error {
	_, err := i.db.Exec(
		ctx,
		updateUserPassword,
		sql.NamedArg{Name: "salt", Value: salt},
		sql.NamedArg{Name: "hashedPassword", Value: hashedPassword},
		sql.NamedArg{Name: "id", Value: userID},
	)
	if err != nil {
		zerologr.Error(err, "Failed to update user password")
	}
	return err
}

// --- impl methods: Organisations ---

// dbGetOrg returns an organisation by ID.
// Returns (nil, errNoOrg) when no matching organisation exists.
func (i *impl) dbGetOrg(ctx context.Context, orgID int64) (*authbasicapi.Organisation, error) {
	rows, err := i.db.Query(
		ctx,
		selectOrg,
		sql.NamedArg{Name: "orgID", Value: orgID},
	)
	if err != nil {
		zerologr.Error(err, "Failed to query organisation")
		return nil, err
	}
	defer rows.Close()

	if !rows.Next() {
		if err := rows.Err(); err != nil {
			zerologr.Error(err, "Failed to iterate organisation rows")
			return nil, err
		}
		return nil, errNoOrg
	}

	var o authbasicapi.Organisation
	if err := rows.Scan(&o.Id, &o.Name); err != nil {
		zerologr.Error(err, "Failed to scan organisation row")
		return nil, err
	}

	return &o, nil
}

func (i *impl) dbListOrgs(ctx context.Context) ([]authbasicapi.Organisation, error) {
	rows, err := i.db.Query(ctx, selectOrgs)
	if err != nil {
		zerologr.Error(err, "Failed to query organisations")
		return nil, err
	}
	defer rows.Close()

	orgs := make([]authbasicapi.Organisation, 0)
	for rows.Next() {
		var o authbasicapi.Organisation
		if err := rows.Scan(&o.Id, &o.Name); err != nil {
			zerologr.Error(err, "Failed to scan organisation row")
			return nil, err
		}
		orgs = append(orgs, o)
	}
	if err := rows.Err(); err != nil {
		zerologr.Error(err, "Failed to iterate organisation rows")
		return nil, err
	}

	return orgs, nil
}

func (i *impl) dbUpdateOrg(ctx context.Context, orgID int64, name string) error {
	_, err := i.db.Exec(
		ctx,
		updateOrg,
		sql.NamedArg{Name: "name", Value: name},
		sql.NamedArg{Name: "orgID", Value: orgID},
	)
	if err != nil {
		zerologr.Error(err, "Failed to update organisation")
	}
	return err
}

func (i *impl) dbDeleteOrg(ctx context.Context, orgID int64) error {
	_, err := i.db.Exec(
		ctx,
		deleteOrg,
		sql.NamedArg{Name: "orgID", Value: orgID},
	)
	if err != nil {
		zerologr.Error(err, "Failed to delete organisation")
	}
	return err
}

// --- impl methods: Groups ---

// dbGetGroup returns a group by ID within an organisation.
// Returns (nil, errNoGroup) when no matching group exists.
func (i *impl) dbGetGroup(ctx context.Context, orgID, groupID int64) (*authbasicapi.Group, error) {
	rows, err := i.db.Query(
		ctx,
		selectGroup,
		sql.NamedArg{Name: "groupID", Value: groupID},
		sql.NamedArg{Name: "orgID", Value: orgID},
	)
	if err != nil {
		zerologr.Error(err, "Failed to query group")
		return nil, err
	}
	defer rows.Close()

	if !rows.Next() {
		if err := rows.Err(); err != nil {
			zerologr.Error(err, "Failed to iterate group rows")
			return nil, err
		}
		return nil, errNoGroup
	}

	var g authbasicapi.Group
	if err = rows.Scan(&g.Id, &g.Name); err != nil {
		zerologr.Error(err, "Failed to scan group row")
		return nil, err
	}

	return &g, nil
}

func (i *impl) dbListGroups(ctx context.Context, orgID int64) ([]authbasicapi.Group, error) {
	rows, err := i.db.Query(
		ctx,
		selectGroups,
		sql.NamedArg{Name: "orgID", Value: orgID},
	)
	if err != nil {
		zerologr.Error(err, "Failed to query groups")
		return nil, err
	}
	defer rows.Close()

	groups := make([]authbasicapi.Group, 0)
	for rows.Next() {
		var g authbasicapi.Group
		if err := rows.Scan(&g.Id, &g.Name); err != nil {
			zerologr.Error(err, "Failed to scan group row")
			return nil, err
		}
		groups = append(groups, g)
	}
	if err := rows.Err(); err != nil {
		zerologr.Error(err, "Failed to iterate group rows")
		return nil, err
	}

	return groups, nil
}

func (i *impl) dbCreateGroup(ctx context.Context, orgID int64, name string) (int64, error) {
	res, err := i.db.Exec(
		ctx,
		insertGroup,
		sql.NamedArg{Name: "name", Value: name},
		sql.NamedArg{Name: "orgID", Value: orgID},
	)
	if err != nil {
		zerologr.Error(err, "Failed to insert group")
		return 0, err
	}

	id, _ := res.LastInsertId()
	return id, nil
}

func (i *impl) dbUpdateGroup(ctx context.Context, orgID, groupID int64, name string) error {
	_, err := i.db.Exec(
		ctx,
		updateGroup,
		sql.NamedArg{Name: "name", Value: name},
		sql.NamedArg{Name: "groupID", Value: groupID},
		sql.NamedArg{Name: "orgID", Value: orgID},
	)
	if err != nil {
		zerologr.Error(err, "Failed to update group")
	}
	return err
}

func (i *impl) dbDeleteGroup(ctx context.Context, orgID, groupID int64) error {
	_, err := i.db.Exec(
		ctx,
		deleteGroup,
		sql.NamedArg{Name: "orgID", Value: orgID},
		sql.NamedArg{Name: "groupID", Value: groupID},
	)
	if err != nil {
		zerologr.Error(err, "Failed to delete group")
	}
	return err
}

// --- impl methods: Group bindings ---

func (i *impl) dbListGroupBindings(
	ctx context.Context,
	orgID, userID int64,
) ([]*models.GroupBinding, error) {
	rows, err := i.db.Query(
		ctx,
		selectGroupBindings,
		sql.NamedArg{Name: "orgID", Value: orgID},
		sql.NamedArg{Name: "userID", Value: userID},
	)
	if err != nil {
		zerologr.Error(err, "Failed to query group bindings")
		return nil, err
	}
	defer rows.Close()

	bindings := make([]*models.GroupBinding, 0)
	for rows.Next() {
		b := &models.GroupBinding{}
		if err := rows.Scan(&b.GroupID, &b.Name); err != nil {
			zerologr.Error(err, "Failed to scan group binding row")
			return nil, err
		}
		bindings = append(bindings, b)
	}
	if err := rows.Err(); err != nil {
		zerologr.Error(err, "Failed to iterate group binding rows")
		return nil, err
	}

	return bindings, nil
}

// --- Transaction helpers ---

// dbCreateOrganisation atomically creates an organisation and its initial admin user.
// Returns the new organisation ID, admin user ID, admin username, and cleartext admin password.
// If the organisation name is already taken, the returned error wraps db.ErrUnique.
//
//nolint:nonamedreturns // welp
func (i *impl) dbCreateOrganisation(
	ctx context.Context,
	name string,
) (orgID, adminUserID int64, adminUsername, adminPassword string, err error) {
	tx, err := i.db.Begin(ctx)
	if err != nil {
		zerologr.Error(err, "Failed to start transaction")
		return 0, 0, "", "", err
	}
	//nolint:errcheck // intentional: no-op if already committed
	defer tx.Rollback()

	zerologr.Info("Creating organisation " + name)
	res, err := tx.Exec(ctx, insertOrg, sql.NamedArg{Name: "name", Value: name})
	if err != nil {
		zerologr.Error(err, "Failed to create organisation")
		return 0, 0, "", "", err
	}
	orgID, _ = res.LastInsertId()
	zerologr.Info(fmt.Sprintf("Created organisation with id %d", orgID))

	adminUsername = fmt.Sprintf("%s-%s", "admin", name)
	adminPassword, salt, hashedAdminPassword := password.Make("")
	res, err = tx.Exec(
		ctx,
		insertUser,
		sql.NamedArg{Name: "name", Value: adminUsername},
		sql.NamedArg{Name: "salt", Value: salt},
		sql.NamedArg{Name: "hashedPassword", Value: hashedAdminPassword},
		sql.NamedArg{Name: "orgID", Value: orgID},
		sql.NamedArg{Name: "isAdmin", Value: true},
	)
	if err != nil {
		zerologr.Error(err, "Failed to create admin user for organisation")
		return 0, 0, "", "", err
	}

	if err = tx.Commit(); err != nil {
		zerologr.Error(err, "Failed to commit organisation creation transaction")
		return 0, 0, "", "", err
	}

	adminUserID, _ = res.LastInsertId()
	return orgID, adminUserID, adminUsername, adminPassword, nil
}

// dbUpdateUserGroupBindings atomically updates a user's group memberships to match desiredGroups.
func (i *impl) dbUpdateUserGroupBindings(
	ctx context.Context,
	orgID, userID int64,
	desiredGroups []string,
) error {
	bindings, err := i.dbListGroupBindings(ctx, orgID, userID)
	if err != nil {
		return err
	}

	toDelete := make([]*models.GroupBinding, 0)
	for _, b := range bindings {
		if !slices.Contains(desiredGroups, b.Name) {
			toDelete = append(toDelete, b)
		}
	}

	tx, err := i.db.Begin(ctx)
	if err != nil {
		zerologr.Error(err, "Failed to start transaction")
		return err
	}
	//nolint:errcheck // intentional: no-op if already committed
	defer tx.Rollback()

	for _, b := range toDelete {
		if _, err := tx.Exec(
			ctx,
			deleteGroupBinding,
			sql.NamedArg{Name: "userID", Value: userID},
			sql.NamedArg{Name: "groupID", Value: b.GroupID},
		); err != nil {
			zerologr.Error(err, "Failed to delete group binding")
			return err
		}
		bindings = slices.DeleteFunc(
			bindings,
			func(gb *models.GroupBinding) bool { return gb.Name == b.Name },
		)
	}

	for _, groupName := range desiredGroups {
		if !slices.ContainsFunc(
			bindings,
			func(b *models.GroupBinding) bool { return b.Name == groupName },
		) {
			if _, err := tx.Exec(
				ctx,
				insertGroupBinding,
				sql.NamedArg{Name: "userID", Value: userID},
				sql.NamedArg{Name: "orgID", Value: orgID},
				sql.NamedArg{Name: "groupName", Value: groupName},
			); err != nil {
				zerologr.Error(err, "Failed to insert group binding")
				return err
			}
		}
	}

	if err := tx.Commit(); err != nil {
		zerologr.Error(err, "Failed to commit group binding transaction")
		return err
	}

	return nil
}
