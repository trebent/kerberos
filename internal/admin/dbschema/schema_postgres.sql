CREATE TABLE IF NOT EXISTS admin_groups (
  id SERIAL PRIMARY KEY,
  name VARCHAR(100) NOT NULL,
  created TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
  updated TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE UNIQUE INDEX IF NOT EXISTS admin_group_name ON admin_groups(name);

CREATE TABLE IF NOT EXISTS admin_permissions (
  id INTEGER PRIMARY KEY,
  name VARCHAR(100) NOT NULL
);

CREATE UNIQUE INDEX IF NOT EXISTS admin_permission_name ON admin_permissions(name);

CREATE TABLE IF NOT EXISTS admin_group_permission_bindings (
  group_id INTEGER,
  permission_id INTEGER,
  FOREIGN KEY(group_id) REFERENCES admin_groups(id) ON DELETE CASCADE ON UPDATE CASCADE,
  FOREIGN KEY(permission_id) REFERENCES admin_permissions(id) ON DELETE CASCADE ON UPDATE CASCADE
);

CREATE UNIQUE INDEX IF NOT EXISTS admin_group_permissions ON admin_group_permission_bindings(group_id, permission_id);

CREATE TABLE IF NOT EXISTS admin_users (
  id SERIAL PRIMARY KEY,
  name VARCHAR(100) NOT NULL,
  superuser BOOLEAN NOT NULL DEFAULT FALSE,
  salt VARCHAR(100) NOT NULL,
  hashed_password VARCHAR(128) NOT NULL,
  created TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
  updated TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE UNIQUE INDEX IF NOT EXISTS admin_user_name ON admin_users(name);

CREATE TABLE IF NOT EXISTS admin_group_bindings (
  user_id INTEGER,
  group_id INTEGER,
  created TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
  updated TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
  FOREIGN KEY(user_id) REFERENCES admin_users(id) ON DELETE CASCADE ON UPDATE CASCADE,
  FOREIGN KEY(group_id) REFERENCES admin_groups(id) ON DELETE CASCADE ON UPDATE CASCADE
);

CREATE UNIQUE INDEX IF NOT EXISTS admin_user_groups ON admin_group_bindings(user_id, group_id);

CREATE TABLE IF NOT EXISTS admin_sessions (
  user_id INTEGER,
  session_id VARCHAR(100),
  expires BIGINT NOT NULL,
  FOREIGN KEY(user_id) REFERENCES admin_users(id) ON DELETE CASCADE ON UPDATE CASCADE
);

CREATE TABLE IF NOT EXISTS admin_debug_sessions (
  id SERIAL PRIMARY KEY,
  backend VARCHAR(100) NOT NULL,
  started_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
  expires_at TIMESTAMPTZ NOT NULL,
  stopped_at TIMESTAMPTZ
);

CREATE TABLE IF NOT EXISTS admin_debug_session_calls (
  id SERIAL PRIMARY KEY,
  session_id INTEGER,
  started_at TIMESTAMPTZ NOT NULL,
  stopped_at TIMESTAMPTZ NOT NULL,
  url VARCHAR(2048),
  method VARCHAR(10),
  status_code INTEGER,
  FOREIGN KEY(session_id) REFERENCES admin_debug_sessions(id) ON DELETE CASCADE ON UPDATE CASCADE
);

CREATE TABLE IF NOT EXISTS admin_debug_session_call_flow_transitions (
  call_id INTEGER,
  component VARCHAR(100),
  direction VARCHAR(10),
  started_at TIMESTAMPTZ NOT NULL,
  stopped_at TIMESTAMPTZ NOT NULL,
  result VARCHAR(20),
  failure_cause VARCHAR(512),
  FOREIGN KEY(call_id) REFERENCES admin_debug_session_calls(id) ON DELETE CASCADE ON UPDATE CASCADE
);

-- Trigger function shared by all tables with an `updated` column.
CREATE OR REPLACE FUNCTION set_updated_timestamp()
RETURNS TRIGGER AS $$
BEGIN
  NEW.updated = CURRENT_TIMESTAMP;
  RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE OR REPLACE TRIGGER admin_group_updated
BEFORE UPDATE ON admin_groups
FOR EACH ROW EXECUTE FUNCTION set_updated_timestamp();

CREATE OR REPLACE TRIGGER admin_user_updated
BEFORE UPDATE ON admin_users
FOR EACH ROW EXECUTE FUNCTION set_updated_timestamp();

CREATE OR REPLACE TRIGGER admin_group_bindings_updated
BEFORE UPDATE ON admin_group_bindings
FOR EACH ROW EXECUTE FUNCTION set_updated_timestamp();
