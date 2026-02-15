CREATE TABLE IF NOT EXISTS groups (
  id INTEGER PRIMARY KEY,
  name VARCHAR(100) NOT NULL,
  created TEXT NOT NULL DEFAULT current_timestamp,
  updated TEXT NOT NULL DEFAULT current_timestamp
);

CREATE UNIQUE INDEX IF NOT EXISTS group_name ON groups(name);

CREATE TABLE IF NOT EXISTS users (
  id INTEGER PRIMARY KEY,
  name VARCHAR(100) NOT NULL,
  salt VARCHAR(100) NOT NULL,
  hashed_password VARCHAR(100) NOT NULL,
  created TEXT NOT NULL DEFAULT current_timestamp,
  updated TEXT NOT NULL DEFAULT current_timestamp
);

CREATE UNIQUE INDEX IF NOT EXISTS user_name ON users(name);

CREATE TABLE IF NOT EXISTS group_bindings (
  user_id INTEGER,
  group_id INTEGER,
  created TEXT NOT NULL DEFAULT current_timestamp,
  updated TEXT NOT NULL DEFAULT current_timestamp,
  FOREIGN KEY(user_id) REFERENCES users(id) ON DELETE CASCADE ON UPDATE CASCADE,
  FOREIGN KEY(group_id) REFERENCES groups(id) ON DELETE CASCADE ON UPDATE CASCADE
);

CREATE UNIQUE INDEX IF NOT EXISTS user_groups ON group_bindings(user_id, group_id);

CREATE TABLE IF NOT EXISTS sessions (
  user_id INTEGER,
  session_id VARCHAR(100),
  expires INTEGER NOT NULL,
  FOREIGN KEY(user_id) REFERENCES users(id) ON DELETE CASCADE ON UPDATE CASCADE
);

CREATE TRIGGER IF NOT EXISTS group_bindings_updated 
AFTER UPDATE ON group_bindings
WHEN old.updated = new.updated
BEGIN
  UPDATE group_bindings SET updated = current_timestamp WHERE user_id = old.user_id AND group_id = old.group_id;
END;

CREATE TRIGGER IF NOT EXISTS user_updated 
AFTER UPDATE ON users
WHEN old.updated = new.updated
BEGIN
  UPDATE users SET updated = current_timestamp WHERE id = old.id;
END;

CREATE TRIGGER IF NOT EXISTS group_updated 
AFTER UPDATE ON groups
WHEN old.updated = new.updated
BEGIN
  UPDATE groups SET updated = current_timestamp WHERE id = old.id;
END;
