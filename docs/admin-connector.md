# Admin Connector

The admin connector is a standalone binary (`cmd/admin-connector`) that sits between a frontend application and the Kerberos admin API. It acts as a reverse proxy that enforces admin session authentication before forwarding requests to the admin API server.

## Purpose

The admin connector solves a common deployment problem: browsers cannot safely forward HTTP-only session cookies set by Kerberos to a backend API that is hosted on a different origin. The connector is deployed at an origin the browser trusts, validates that the incoming request carries a valid admin session cookie, and then proxies the request to the upstream admin API.

```
Browser → Admin Connector (same origin or allowed CORS origin) → Kerberos Admin API
```

The connector:

1. Reads the `krb-admin-session` cookie from the incoming request.
2. Queries the same persistence store as the Kerberos admin API to verify the session exists and has not expired.
3. Forwards the request to the configured upstream target if the session is valid.
4. Returns `401 Unauthorized` if the session is missing or expired.

---

## Configuration

The connector is configured via a single JSON file passed with the `--config` flag, and a set of environment variables.

### Config File

```json
{
  "persistence": {
    "driver": "sqlite",
    "address": "krb.db"
  }
}
```

All fields except `persistence` are optional.

#### `persistence` (required)

Must point to the same database as the Kerberos admin API so the connector can validate sessions.

```json
"persistence": {
  "driver": "sqlite",
  "address": "/data/krb.db"
}
```

For PostgreSQL:

```json
"persistence": {
  "driver": "postgres",
  "address": "localhost:5432",
  "postgres": {
    "database": "kerberos",
    "username": "krb",
    "password": "secret",
    "sslMode": "require"
  }
}
```

#### `tls` (optional)

Configures TLS for the connector's own listening server.

```json
"tls": {
  "serverCertFile": "/certs/connector.pem",
  "serverKeyFile": "/certs/connector-key.pem"
}
```

When omitted, the connector listens on plain HTTP.

#### `targetTls` (optional)

Configures TLS for the upstream connection to the Kerberos admin API.

```json
"targetTls": {
  "rootCAFile": "/certs/admin-ca.pem",
  "insecureSkipVerify": false
}
```

| Field | Description |
|---|---|
| `rootCAFile` | Path to a PEM-encoded CA bundle used to verify the admin API's certificate. When omitted, the system certificate pool is used. |
| `insecureSkipVerify` | Disables server certificate verification. Use only in non-production environments. |

When `targetTls` is omitted, the connector proxies to the upstream using plain HTTP.

#### `origins` (optional)

Controls CORS and origin filtering for requests the connector receives from browsers. The same three options as the rest of Kerberos apply:

```json
"origins": {
  "allowedOrigins": ["https://admin.example.com"]
}
```

| Field | Description |
|---|---|
| `allowedOrigins` | List of specific allowed origins. Mutually exclusive with `allowAll`. |
| `allowAll` | When `true`, the `Access-Control-Allow-Origin` header echoes back whatever `Origin` was received. Mutually exclusive with `allowedOrigins`. |
| `denyAll` | When `true`, any request with an `Origin` header is rejected with `403`. Mutually exclusive with `allowedOrigins` and `allowAll`. |

---

### Environment Variables

The connector is also configured through environment variables. These apply on top of (or instead of) the config file fields and control the runtime behaviour of the binary itself.

| Variable | Default | Description |
|---|---|---|
| `TARGET` | *(required)* | Host and port of the upstream Kerberos admin API, e.g. `localhost:9090`. |
| `PORT` | `30100` | Port on which the connector listens. |
| `READ_TIMEOUT_SECONDS` | `5` | HTTP server read timeout in seconds. |
| `WRITE_TIMEOUT_SECONDS` | `5` | HTTP server write timeout in seconds. |
| `OBSERVABILITY_ENABLED` | `true` | Enables or disables OpenTelemetry instrumentation. |
| `RUNTIME_METRICS` | `true` | When true, exposes Go runtime metrics via OpenTelemetry. |
| `LOG_TO_CONSOLE` | `false` | When true, logs are written in a human-readable console format. |
| `LOG_VERBOSITY` | `0` | Increases log verbosity. Higher values emit more detail. |
| `VERSION` | `unset` | Sets the service version reported in telemetry. |

---

## Observability

When observability is enabled, the connector emits the following OpenTelemetry metrics:

| Metric | Description |
|---|---|
| `admin_connector_calls_total` | Total number of requests forwarded to the admin API. |
| `admin_connector_calls_denied_total` | Total number of requests rejected due to missing or expired sessions. |
| `admin_connector_callout_failures_total` | Total number of errors encountered while proxying to the upstream admin API. |

Each request also generates an OpenTelemetry span (server kind) containing the HTTP method and URL.

---

## Minimal Example

```json
{
  "persistence": {
    "driver": "sqlite",
    "address": "/data/krb.db"
  },
  "origins": {
    "allowedOrigins": ["https://admin.example.com"]
  }
}
```

Start with:

```sh
TARGET=localhost:9090 ./admin-connector --config connector.json
```

---

## Annotated Production Example

```json
{
  "persistence": {
    "driver": "postgres",
    "address": "db.internal:5432",
    "postgres": {
      "database": "kerberos",
      "username": "krb",
      "password": "secret",
      "sslMode": "require"
    }
  },
  "tls": {
    "serverCertFile": "/certs/connector.pem",
    "serverKeyFile": "/certs/connector-key.pem"
  },
  "targetTls": {
    "rootCAFile": "/certs/internal-ca.pem"
  },
  "origins": {
    "allowedOrigins": ["https://admin.example.com"]
  }
}
```

```sh
TARGET=admin-api.internal:9090 \
PORT=443 \
VERSION=1.2.3 \
  ./admin-connector --config connector.json
```
