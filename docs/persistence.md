# Persistence

Kerberos keeps its administrative state, such as users, groups, permissions, login sessions and debug captures, in a SQL database. You can use a SQLite file, which needs nothing else installed, or a PostgreSQL server. This page covers what Kerberos stores, how to configure each driver and what to plan for when you run it in production.

## When you need a database

Every Kerberos instance uses a database, even one that only forwards traffic. The admin API is always on, and at startup it creates its tables and makes sure the superuser account exists. If you leave out the `persistence` section, Kerberos uses a SQLite file in its working directory (see [Key reference](#key-reference) for the defaults). You choose which database to use and where its data lives.

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

Configuration, OpenAPI specifications, metrics, traces and forwarded traffic are kept elsewhere. Debug sessions record the URL, method, status code and timings of the calls they capture, without request or response bodies.

## Choosing a driver

| | SQLite (`sqlite`) | PostgreSQL (`postgres`) |
| --- | --- | --- |
| What you need | Nothing extra. Data lives in a single file. | A PostgreSQL server you run and maintain. |
| Number of Kerberos instances | One. | Several instances can share one database. |
| Good for | Local development, demos, single-host deployments with a persistent volume. | Production, high availability, managed or backed-up databases. |

Use SQLite to get started or when one gateway instance is enough. Use PostgreSQL when you run more than one replica, when your platform has no lasting local storage, or when your team already runs PostgreSQL with backups and monitoring. Before you scale out, also read the [scaling notes](./deployment.md#scaling) in Deployment.

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

`address` is the path to the database file. Kerberos creates the file if it does not exist. The directory must already exist, and Kerberos must be allowed to write to it. A relative path is resolved against the process's working directory, not against the config file.

You may add SQLite connection parameters to the path after `?`. Kerberos adds `_pragma=foreign_keys=on` itself so that deletes cascade, so leave that setting as it is.

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

Keep the password out of the config file with an `${env:...}` reference. For the reference syntax, see [Configuration](./configuration.md#references).

Create the database (and the user, if you use one) before Kerberos starts. Kerberos creates the tables inside it. The user needs permission to create tables, indexes and triggers in that database.

### Key reference

| Key | Type | Default | Description |
| --- | --- | --- | --- |
| `driver` | string, `sqlite` or `postgres` | `sqlite` (when the section is omitted) | Database engine. |
| `address` | string, at least 1 character | `krb.db` (when the section is omitted) | For `sqlite`, the path to the database file. For `postgres`, the server as `host:port`. |
| `postgres` | object | none | PostgreSQL settings. Required when `driver` is `postgres`, ignored for `sqlite`. |
| `postgres.database` | string, at least 1 character | none (required) | Name of an existing database. |
| `postgres.username` | string | none | Database user. Set it together with `password`. |
| `postgres.password` | string | none | Database password. Set it together with `username`. |
| `postgres.sslMode` | string | `require` | PostgreSQL SSL mode, such as `disable`, `require`, `verify-ca` or `verify-full`. |

The `driver` and `address` defaults apply when the whole `persistence` section is left out. If you include the section, set both.

## Table creation and upgrades

At startup, Kerberos creates every table, index and trigger it needs, if they are not there yet:

- The admin tables are always created.
- The organisation, user, group and session tables for basic authentication are created when basic authentication is enabled in the `auth` section.

If table creation fails, Kerberos logs the error and exits.

Before you upgrade:

- **Read the release notes** for any database changes in the new version.
- **Take a backup**, so you can restore the previous state if you need to.

A new version creates any new tables it needs. Each driver has its own tables, so when you switch between SQLite and PostgreSQL, the new database starts empty and a new superuser is created from the config.

## Running in production

### SQLite

- **Use a persistent volume.** In a container, mount a volume and point `address` at an absolute path inside it, such as `/krbdata/krb.db`, so users, groups and sessions are kept when the container is recreated.
- **Make it writable.** The official image runs as a non-root user, so that user must be able to write to the volume directory and the file. The Docker Compose setup that [krbctl](./krbctl.md) generates uses a small start-up container that creates `/krbdata/krb.db` and sets the file permissions.
- **Run a single instance**, with the file on a local disk.

### PostgreSQL

- **Write `address` as `host:port`**, for example `postgres.internal:5432`. For an IPv6 server, use a host name that resolves to it.
- **Set `username` and `password` together.**
- **Set `sslMode`.** Use `verify-full` in production, and `disable` only on trusted local networks.
- **Use plain values** for credentials and names, without spaces, quotes or backslashes.

### Start the database first

Kerberos connects to the database at startup, so start the database before Kerberos. Use your platform's tools for this, for example `restart: on-failure` together with `depends_on: condition: service_healthy` in Docker Compose, or a restart policy and a database readiness check in Kubernetes. Kerberos keeps running if the database goes away for a while, and requests that need it (logins, session checks, admin API calls) work again once it is back.

## Sharing the database with admin-connector

[admin-connector](./admin-connector.md) checks admin login sessions by reading the same database that Kerberos writes to. Give it a `persistence` section that is identical to the Kerberos one:

- For SQLite, both processes open the same file, so run them on the same host with the same volume mounted. The generated Docker Compose setup mounts a shared `krbdata` volume into both containers.
- For PostgreSQL, point both at the same server and database.

Kerberos creates the tables, so start Kerberos against the database once before you start the connector.

## Backups

The database holds every account, group, permission binding and active login in your installation. Back it up like any other stateful service.

- **SQLite:** Stop Kerberos and copy the database file, or use SQLite's online backup (for example, `sqlite3 krb.db ".backup backup.db"`) from a host or helper container that has the `sqlite3` tool.
- **PostgreSQL:** Use your usual tooling, such as `pg_dump`, managed-service snapshots or continuous archiving.

To restore, put the data back before you start Kerberos. A restored database keeps the superuser credentials it had when it was backed up.

## Things to know

- **The superuser is created on the first start.** See [Admin API: the superuser](./admin-api.md#the-superuser) for the defaults and how to change the password. Change the default secret before you expose the admin port.
- **Sessions are removed on logout and refresh.**
- **Debug data stays until you delete it.** Recorded calls are kept until you delete the debug session that owns them. See [Debug sessions](./admin-debugging.md).

## Related docs

- [Configuration](./configuration.md): config file format and `${env:...}` references
- [Deployment](./deployment.md): running Kerberos in production, scaling, restart policies
- [Admin API](./admin-api.md): superuser, admin users, groups and permissions
- [Organisations](./organizations.md): basic-auth organisations, users and groups
- [Sessions](./sessions.md): login sessions and cookies
- [Admin debugging](./admin-debugging.md): debug sessions and captured calls
- [admin-connector](./admin-connector.md): protecting internal tools with admin login
- [krbctl](./krbctl.md): generating a Docker Compose setup with SQLite or PostgreSQL
