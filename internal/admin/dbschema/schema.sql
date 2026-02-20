CREATE TABLE IF NOT EXISTS admin_groups (
  id INTEGER PRIMARY KEY,
  name VARCHAR(100) NOT NULL,
  created TEXT NOT NULL DEFAULT current_timestamp,
  updated TEXT NOT NULL DEFAULT current_timestamp
);

CREATE UNIQUE INDEX IF NOT EXISTS admin_group_name ON admin_groups(name);

CREATE TABLE IF NOT EXISTS admin_users (
  id INTEGER PRIMARY KEY,
  name VARCHAR(100) NOT NULL,
  salt VARCHAR(100) NOT NULL,
  hashed_password VARCHAR(100) NOT NULL,
  created TEXT NOT NULL DEFAULT current_timestamp,
  updated TEXT NOT NULL DEFAULT current_timestamp
);

CREATE UNIQUE INDEX IF NOT EXISTS admin_user_name ON admin_users(name);

CREATE TABLE IF NOT EXISTS admin_group_bindings (
  user_id INTEGER,
  group_id INTEGER,
  created TEXT NOT NULL DEFAULT current_timestamp,
  updated TEXT NOT NULL DEFAULT current_timestamp,
  FOREIGN KEY(user_id) REFERENCES admin_users(id) ON DELETE CASCADE ON UPDATE CASCADE,
  FOREIGN KEY(group_id) REFERENCES admin_groups(id) ON DELETE CASCADE ON UPDATE CASCADE
);

CREATE UNIQUE INDEX IF NOT EXISTS admin_user_groups ON admin_group_bindings(user_id, group_id);

CREATE TABLE IF NOT EXISTS admin_sessions (
  user_id INTEGER,
  session_id VARCHAR(100),
  expires INTEGER NOT NULL,
  FOREIGN KEY(user_id) REFERENCES admin_users(id) ON DELETE CASCADE ON UPDATE CASCADE
);

CREATE TRIGGER IF NOT EXISTS admin_group_bindings_updated 
AFTER UPDATE ON admin_group_bindings
WHEN old.updated = new.updated
BEGIN
  UPDATE admin_group_bindings SET updated = current_timestamp WHERE user_id = old.user_id AND group_id = old.group_id;
END;

CREATE TRIGGER IF NOT EXISTS admin_user_updated 
AFTER UPDATE ON admin_users
WHEN old.updated = new.updated
BEGIN
  UPDATE admin_users SET updated = current_timestamp WHERE id = old.id;
END;

CREATE TRIGGER IF NOT EXISTS admin_group_updated 
AFTER UPDATE ON admin_groups
WHEN old.updated = new.updated
BEGIN
  UPDATE admin_groups SET updated = current_timestamp WHERE id = old.id;
END;
