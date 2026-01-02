CREATE TABLE IF NOT EXISTS organisations (
  id INTEGER PRIMARY KEY,
  name VARCHAR(100) NOT NULL
);

CREATE UNIQUE INDEX IF NOT EXISTS organisation_name ON organisations(name);

CREATE TABLE IF NOT EXISTS groups (
  id INTEGER PRIMARY KEY,
  name VARCHAR(100) NOT NULL
);

CREATE UNIQUE INDEX IF NOT EXISTS group_name ON groups(name);

CREATE TABLE IF NOT EXISTS users (
  id INTEGER PRIMARY KEY,
  name VARCHAR(100) NOT NULL,
  organisation INTEGER,
  FOREIGN KEY(organisation) REFERENCES organisations(id)
);

CREATE UNIQUE INDEX IF NOT EXISTS user_name ON users(name);

PRAGMA foreign_keys=ON;
