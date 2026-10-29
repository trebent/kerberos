# Persistence

Kerberos keeps its administrative state, such as users, groups, permissions, login sessions and debug captures, in a SQL database. You can use an embedded SQLite file, which needs no extra infrastructure, or an external PostgreSQL server. This page covers what Kerberos stores, how to configure each driver and what to plan for when you run it in production.

## When you need a database

Every Kerberos instance uses a database, even one that only forwards traffic. The admin API is always loaded, and at startup it creates its tables and makes sure the superuser account exists. If you leave out the `persistence` section, Kerberos uses a SQLite file named `krb.db` in its working directory.

Choosing a database is therefore not optional. What you choose is which database to use and where its data lives.

## What is stored

| Data | Created when | Notes |
| --- | --- | --- |
| Superuser account | Every startup, if no superuser exists yet | Created from `admin.superUser` in the config. See [Things to know](#things-to-know). |
| Admin users, admin groups, group memberships | Through the [admin API](./admin-api.md) | Passwords are stored salted and hashed. |
| Admin permissions and group-to-permission bindings | Permissions are inserted at every startup if missing; bindings come from the admin API | The fixed permission names are listed in [Admin API](./admin-api.md). |
| Admin login sessions | Superuser or admin user login | Session and refresh identifiers with an expiry time. See [Sessions](./sessions.md). |
| Debug sessions, captured calls and their flow transitions | Through the debug endpoints of the admin API | See [Admin debugging](./admin-debugging.md). |
| Organisations, users, groups, group bindings | Through the basic-auth API | Only when basic authentication is enabled. See [Organisations](./organizations.md). |
| Basic-auth login sessions | Organisation user login | Only when basic authentication is enabled. |

Kerberos does not store any configuration, OpenAPI specifications, metrics, traces or the traffic it forwards. Debug sessions are the exception: they record the URL, method, status code and timings of the calls they capture, but not request or response bodies.

## Choosing a driver

| | SQLite (`sqlite`) | PostgreSQL (`postgres`) |
| --- | --- | --- |
| Extra infrastructure | None. Data lives in a single file. | A PostgreSQL server you run and maintain. |
| Number of Kerberos instances | One. The file cannot be shared between hosts. | Several instances can share one database. |
| Concurrency | Kerberos uses one connection to the file, so database access is serialised. | Uses a connection pool. |
| Good for | Local development, demos, single-host deployments with a persistent volume. | Production, high availability, managed or backed-up databases. |

Use SQLite to get started or when one gateway instance is enough. Use PostgreSQL when you run more than one replica, when your platform cannot provide a durable local volume, or when your team already operates PostgreSQL with backups and monitoring. Before you scale out, also read the scaling notes in [Deployment](./deployment.md). Some admin state, such as which backends are currently being debugged, is held in each process's memory and is not shared through the database.

## Configuration

The `persistence` section is optional at the top level of the config file. If you include it, `driver` and `address` are required. If `driver` is `postgres`, the `postgres` object and its `database` key are also required.

The authoritative schema is [`internal/config/schemas/persistence_schema.json`](../internal/config/schemas/persistence_schema.json).

### SQLite

```json
{
  "persistence": {
    "driver": "sqlite",
    "address": "/var/lib/kerberos/krb.db"
  }
}
```

`address` is the path to the database file. If the file does not exist, Kerberos creates it, but the directory must already exist and the Kerberos process must be allowed to write to it. A relative path is resolved against the process's working directory, not against the config file.

You may append SQLite connection parameters to the path after `?`. Kerberos always adds `_pragma=foreign_keys=on` itself, so deletes cascade correctly. Do not override that pragma.

### PostgreSQL

```json
{
  "persistence": {
    "driver": "postgres",
    "address": "postgres.internal:5432",
    "postgres": {
      "database": "kerberos",
      "username": "kerberos",
      "password": "${env:KRB_DB_PASSWORD}",
      "sslMode": "verify-full"
    }
  }
}
```

For a local PostgreSQL without TLS, such as a Docker Compose stack, set `"sslMode": "disable"`:

```json
{
  "persistence": {
    "driver": "postgres",
    "address": "postgres:5432",
    "postgres": {
      "database": "kerberos",
      "username": "kerberos",
      "password": "${env:KRB_DB_PASSWORD}",
      "sslMode": "disable"
    }
  }
}
```

Keep the password out of the config file with an `${env:...}` reference. For the reference syntax and its fallbacks, see [Configuration](./configuration.md).

Create the database (and the user, if you use one) before Kerberos starts. Kerberos creates tables but not the database itself. The user needs permission to create tables, indexes and triggers in that database.

### Key reference

| Key | Type | Default | Description |
| --- | --- | --- | --- |
| `driver` | string, `sqlite` or `postgres` | `sqlite` (when the section is omitted) | Database engine. |
| `address` | string, at least 1 character | `krb.db` (when the section is omitted) | For `sqlite`, the path to the database file. For `postgres`, the server as `host:port`. |
| `postgres` | object | none | PostgreSQL settings. Required when `driver` is `postgres`, ignored for `sqlite`. |
| `postgres.database` | string, at least 1 character | none (required) | Name of an existing database. |
| `postgres.username` | string | none | Database user. Used only if `password` is also set. |
| `postgres.password` | string | none | Database password. Used only if `username` is also set. |
| `postgres.sslMode` | string | not sent; the driver's default (`require`) applies | PostgreSQL SSL mode, such as `disable`, `require`, `verify-ca` or `verify-full`. |

The defaults apply only when the entire `persistence` section is left out. A partial section fails validation.

## Table creation and upgrades

At startup, Kerberos creates every table, index and trigger it needs, using "create if not exists" statements:

- The admin tables are always created.
- The organisation, user, group and session tables for basic authentication are created only when basic authentication is enabled in the `auth` section.

Each schema step has a 5-second time limit. If table creation fails, Kerberos logs the error and exits.

Kerberos has no migration framework and does not track a schema version. In practice this means:

- Starting a new version against an existing database creates any **new** tables that version introduces and leaves existing tables unchanged.
- If a release changes the definition of an **existing** table, for example by adding a column, Kerberos does not alter that table. Read the release notes before you upgrade, and apply any required changes yourself.
- Downgrading is not supported. Take a backup before every upgrade so that you can restore the previous state.

Each driver has its own set of schema statements. Kerberos cannot convert data from SQLite to PostgreSQL, or the other way round. If you switch drivers, the new database starts empty and a new superuser is created from the config.

## Running in production

### SQLite

- **Use a persistent volume.** In a container, mount a volume and point `address` at an absolute path inside it, such as `/krbdata/krb.db`. The default `krb.db` is resolved against the container's working directory, which is not persistent, so all users, groups and sessions are lost when the container is recreated.
- **Make it writable.** The official image runs as a non-root user. The volume directory, and the file if it already exists, must be writable by that user. The Docker Compose setup that [krbctl](./krbctl.md) generates uses a small init container that creates `/krbdata/krb.db` and sets permissive file modes for this purpose.
- **Run a single instance.** Do not run several Kerberos replicas against one SQLite file, and do not put the file on a network file system.

### PostgreSQL

- **`address` must be exactly `host:port`.** Kerberos splits the value on `:`, so a missing port, a URL such as `postgres://...`, or an IPv6 literal makes startup fail. For an IPv6 server, use a hostname that resolves to it.
- **Set credentials as a pair.** `username` and `password` are sent only when both are present. If you set only one of them, Kerberos connects without credentials and the driver falls back to its defaults, such as the operating system user name.
- **Set `sslMode` explicitly.** If you leave it out, the PostgreSQL driver defaults to `require`, so a server without TLS rejects the connection. Use `disable` only on trusted local networks, and prefer `verify-full` in production.
- **Avoid spaces, quotes and backslashes in values.** Connection values are passed to the driver without quoting, so a password, user name or database name that contains whitespace, `'` or `\` may not be parsed correctly.

### Startup when the database is unavailable

Kerberos connects to the database once, at startup. If it cannot reach the database, it exits right away instead of retrying. For PostgreSQL, this means the gateway can fail if it starts before the database is ready.

Use your platform's restart mechanisms to handle this. Examples are `restart: on-failure` together with `depends_on: condition: service_healthy` in Docker Compose, or the default restart policy and a database readiness check in Kubernetes. If the database becomes unavailable while Kerberos is running, requests that need it (logins, session checks, admin API calls) fail until it is back. The process keeps running.

## Sharing the database with admin-connector

[admin-connector](./admin-connector.md) checks admin login sessions by reading the same database that Kerberos writes to. Give it a `persistence` section that is identical to the Kerberos one:

- For SQLite, both processes must be able to open the same file. In practice, they must run on the same host with the same volume mounted. The generated Docker Compose setup mounts a shared `krbdata` volume into both containers.
- For PostgreSQL, point both at the same server and database.

admin-connector does not create tables. Start Kerberos against the database at least once before you start the connector.

## Backups

The database holds every account, group, permission binding and active login in your installation. Back it up like any other stateful service.

- **SQLite:** Back up the database file. Copying a file that is being written to can capture an inconsistent state, so stop Kerberos first, or use SQLite's online backup (for example, `sqlite3 krb.db ".backup backup.db"`) from a host or sidecar that has the `sqlite3` tool. The Kerberos image does not include it.
- **PostgreSQL:** Use your usual tooling, such as `pg_dump`, managed-service snapshots or continuous archiving.

To restore, put the data back before you start Kerberos. Because the superuser is created only when none exists, a restored database keeps the superuser credentials it had when it was backed up.

## Things to know

- **The superuser is created only once.** On first startup, Kerberos creates the superuser from `admin.superUser` (`admin` / `secret` if not configured). After that, changing the client ID or secret in the config has no effect. Change the password through the admin API instead, as described in [Admin API](./admin-api.md). Change the default secret before you expose the admin port.
- **Expired sessions are not purged.** Session rows are removed on logout and when a session is replaced during refresh. Sessions that simply expire stay in the table. This uses little space, but the tables grow over time.
- **Debug data stays until you delete it.** Captured calls are kept until you delete the debug session that owns them. When you restart Kerberos, the debug sessions stay in the database, but capture does not resume, because the set of backends being debugged is held in memory.
- **No connection tuning.** There are no settings for pool size, connection timeouts or statement timeouts.

## Related docs

- [Configuration](./configuration.md): config file format and `${env:...}` references
- [Deployment](./deployment.md): running Kerberos in production, scaling, restart policies
- [Admin API](./admin-api.md): superuser, admin users, groups and permissions
- [Organisations](./organizations.md): basic-auth organisations, users and groups
- [Sessions](./sessions.md): login sessions and cookies
- [Admin debugging](./admin-debugging.md): debug sessions and captured calls
- [admin-connector](./admin-connector.md): protecting internal tools with admin login
- [krbctl](./krbctl.md): generating a Docker Compose setup with SQLite or PostgreSQL
