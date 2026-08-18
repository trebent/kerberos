# Admin Connector

The admin connector is a standalone binary (`cmd/admin-connector`) that acts as an authenticating reverse proxy. It validates that incoming requests carry a valid Kerberos admin session cookie and, if so, forwards the request to any configured upstream service.

## Purpose

The admin connector is used when a service should only be accessible to users who hold a valid Kerberos admin session. The connector reads the session cookie from each incoming request, checks it against the Kerberos admin session store, and either proxies the request to the configured target or rejects it with `401 Unauthorized`.

```
Client → Admin Connector (validates admin session cookie) → Target service
```

The connector:

1. Reads the session cookie from the incoming request.
2. Queries the Kerberos admin persistence store to verify the session exists and has not expired.
3. Forwards the request to the configured upstream target if the session is valid.
4. Returns `401 Unauthorized` if the session is missing or expired.

---

## Configuration

The connector is configured via a single JSON file passed with the `--config` flag, and a set of environment variables.

### Config File

The config file supports the following high-level sections:

- **`persistence`** (required) — points to the same database used by Kerberos (SQLite or PostgreSQL) so the connector can validate sessions.
- **`tls`** (optional) — configures TLS for the connector's own listening server.
- **`targetTls`** (optional) — configures TLS for the outbound connection to the target.
- **`origins`** (optional) — controls CORS / origin filtering for browser clients.

---

### Environment Variables

The connector is also configured through environment variables. These apply on top of (or instead of) the config file fields and control the runtime behaviour of the binary itself.

| Variable | Default | Description |
|---|---|---|
| `TARGET` | *(required)* | Host and port of the upstream service to forward authenticated requests to, e.g. `my-service:8080`. |
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
| `admin_connector_calls_total` | Total number of requests forwarded to the target. |
| `admin_connector_calls_denied_total` | Total number of requests rejected due to missing or expired sessions. |
| `admin_connector_callout_failures_total` | Total number of errors encountered while proxying to the target. |

Each request also generates an OpenTelemetry span (server kind) containing the HTTP method and URL.



