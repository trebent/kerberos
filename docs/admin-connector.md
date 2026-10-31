# Admin connector

The admin connector (`admin-connector`) is a small reverse proxy that you put in front of an internal tool, such as the Jaeger UI or a Grafana instance. It lets a request through only if it carries a valid Kerberos Admin API login session, so the people who can log in to the Admin API can reach the tool, and nobody else can.

## Use cases

- Expose the Jaeger UI, a Prometheus UI or another tool with no login of its own, without making it public.
- Reuse the admin logins you already manage in Kerberos instead of setting up separate credentials for each tool.
- Put an operator-only page behind the same login as the Admin API.

The connector is not part of the gateway. It runs as its own process (or container), listens on its own port (`PORT`, see [Environment variables](#environment-variables)) and forwards to exactly one target. To protect several tools, run one connector per tool.

## How it works

```text
Browser ──(session cookie)──> admin-connector :30100 ──> internal tool (TARGET)
                                     │
                                     └── looks up the session in the Kerberos database
```

For every request, the connector:

1. Handles origins first, if `origins` is configured (see [Origins](#origins)). A browser's preflight request (`OPTIONS` with an allowed `Origin`) is answered with `204` here and needs no session.
2. Reads the `session` cookie. If there is none, it answers `401`.
3. Looks the session up directly in the Kerberos database. It accepts the request only if the value matches an **Admin API session** (an admin user or the superuser) that has not expired. Otherwise it answers `401`.
4. Forwards the request to `TARGET` and returns the target's response as is.

Things to know about the check:

- **Admin logins only.** Any valid admin user or superuser session is accepted. Sessions from organisation (basic-auth) logins get `401`.
- **Shared database.** The connector reads sessions from the Kerberos database, so it must use the same database as Kerberos (see [Persistence](#persistence)).
- **Sessions are refreshed on the Admin API.** When an admin session expires, the user refreshes it or logs in again on the Admin API. See [Sessions](./sessions.md).

### What is forwarded

The request reaches the target with the same method, path, query string, body and headers, including the `Cookie` header. Differences from the original request:

- The `Host` header is set to the target's host.
- Incoming `X-Forwarded-For`, `X-Forwarded-Host`, `X-Forwarded-Proto` and `Forwarded` headers are removed.
- When observability is enabled, trace context headers (`traceparent`) are added so the target can join the trace.

The connector reaches the target over HTTP, so keep the connector and the target on the same private network.

### Responses from the connector

When the connector itself answers, it uses the same JSON error format as the Kerberos APIs:

| Status | When | Body |
| --- | --- | --- |
| `204` | CORS preflight from an allowed origin | empty |
| `401` | No `session` cookie, unknown session, organisation user session, or expired session | `{"errors":["Unauthorized"]}` |
| `403` | `Origin` header not allowed by `origins` | `{"errors":["Forbidden"]}` |
| `502` | The target could not be reached or the proxied request failed | `{"errors":["Bad Gateway"]}` |

All other responses come from the target.

## Getting the session cookie

Users log in on the Admin API (on the Kerberos admin port), and the browser then sends the resulting `session` cookie to the connector:

1. Open the admin login in the browser on the **same host name** that the connector uses, for example `https://ops.example.com:30001` for a connector at `https://ops.example.com:30100`.
2. Log in as an admin user (`POST /api/admin/login`) or as the superuser (`POST /api/admin/superuser/login`).
3. Open the connector URL, for example `https://ops.example.com:30100/`.

This works because browsers do not separate cookies by port: a cookie set by `ops.example.com:30001` is also sent to `ops.example.com:30100`. Keep these points in mind:

- **Different host names.** If the connector has its own host name (for example `jaeger.example.com` while the Admin API is at `admin.example.com`), set `admin.api.cookies.domain` in the Kerberos configuration to a shared parent domain such as `example.com`. See [Cookie configuration](./sessions.md#cookie-configuration).
- **HTTPS.** Kerberos cookies are always `Secure`, so browsers only send them over HTTPS, except to `localhost`. Serve the connector over HTTPS (with the `tls` block or a proxy that handles HTTPS) unless you reach it as `localhost`. See [The Secure flag needs HTTPS](./sessions.md#the-secure-flag-needs-https).
- **Use one login per browser.** Admin and organisation logins use the same `session` cookie name, so log in as an admin user in the browser you use for the connector. See [Keep admin and organisation logins apart](./sessions.md#keep-admin-and-organisation-logins-apart).

From the command line, use a cookie jar:

```bash
curl -c admin.jar -X POST http://localhost:30001/api/admin/superuser/login \
  -H 'Content-Type: application/json' \
  -d '{"clientId":"admin","clientSecret":"secret"}'

curl -b admin.jar http://localhost:30100/
```

## Running the connector

The connector needs a configuration file, passed with `--config`, and the `TARGET` environment variable:

```bash
TARGET=jaeger:16686 admin-connector --config /connector.json
```

Without `--config`, it prints its usage and the list of environment variables and exits with status 1. It also exits at startup if an environment variable or the configuration file is invalid, or if it cannot open the database.

### Container image

The image is `ghcr.io/trebent/kerberos/admin-connector`. Tags:

| Tag | Contents |
| --- | --- |
| `<release tag>` | A specific release. |
| `latest` | The most recent release. |
| `unstable` | The latest build from the `main` branch. |

The image is built on a minimal base image, runs as a non-root user, exposes port 30100 and uses `/connector` as its entrypoint, so you only pass the flags:

```bash
docker run --rm -p 30100:30100 \
  -e TARGET=jaeger:16686 \
  -v "$PWD/connector.json:/connector.json:ro" \
  ghcr.io/trebent/kerberos/admin-connector:latest --config /connector.json
```

The image sets `VERSION` to the image version.

### Binary

Build it from source with `make connector/build`, which writes `build/connector`, or with `go build ./cmd/admin-connector`.

## Environment variables

Environment variables and the configuration file control different things, and they do not overlap. Use environment variables for where the connector listens, what it forwards to, timeouts, logging and telemetry. Use the configuration file for the database, origins and TLS.

| Variable | Default | Description |
| --- | --- | --- |
| `TARGET` | required | The tool to forward to, as `host:port` (for example `jaeger:16686`) or just `host` (port 80). The port must be between 1000 and 65535. No scheme or path. |
| `PORT` | `30100` | Port the connector listens on. Must be between 1000 and 65535. |
| `READ_TIMEOUT_SECONDS` | `5` | Maximum time to read a whole request, including the body. Must be greater than 0. |
| `WRITE_TIMEOUT_SECONDS` | `5` | Maximum time to write the response, which includes waiting for the target. Must be greater than 0. |
| `OBSERVABILITY_ENABLED` | `true` | Turns OpenTelemetry metrics and tracing on or off. |
| `RUNTIME_METRICS` | `true` | Adds Go runtime metrics (memory, goroutines and so on) when observability is enabled. |
| `LOG_TO_CONSOLE` | `false` | `true` for human-readable logs. Otherwise logs are JSON. |
| `LOG_VERBOSITY` | `0` | Higher values log more detail. Must be 0 or greater. |
| `VERSION` | `unset` | Version reported in logs and telemetry. |

The standard OpenTelemetry variables select where telemetry goes, the same as for Kerberos itself, for example `OTEL_EXPORTER_OTLP_ENDPOINT`, `OTEL_EXPORTER_OTLP_TRACES_PROTOCOL`, `OTEL_TRACES_EXPORTER`, `OTEL_METRICS_EXPORTER`, `OTEL_EXPORTER_PROMETHEUS_HOST` and `OTEL_EXPORTER_PROMETHEUS_PORT`. See [Observability](./observability.md).

## Configuration file

The configuration file is JSON. Its schema is [`internal/config/schemas/connector_schema.json`](../internal/config/schemas/connector_schema.json). Unknown keys are rejected.

```json
{
  "persistence": {
    "driver": "postgres",
    "address": "postgres:5432",
    "postgres": {
      "database": "kerberos",
      "username": "kerberos",
      "password": "kerberos",
      "sslMode": "disable"
    }
  },
  "origins": {
    "denyAll": true
  },
  "tls": {
    "serverCertFile": "/certs/connector.pem",
    "serverKeyFile": "/certs/connector-key.pem"
  }
}
```

| Key | Type | Required | Description |
| --- | --- | --- | --- |
| `persistence` | object | yes | The Kerberos database. Must be the same database Kerberos uses. |
| `origins` | object | no | Which browser origins may call the connector. Not set means no origin handling. |
| `tls` | object | no | HTTPS for the connector's own port. Not set means plain HTTP. |

### Persistence

The `persistence` block has the same format as in the Kerberos configuration, and it should be a copy of it:

```json
{
  "persistence": {
    "driver": "sqlite",
    "address": "/krbdata/krb.db"
  }
}
```

- With SQLite, the connector and Kerberos must open the same file, which in practice means the same host and a shared volume.
- With PostgreSQL, point both at the same server and database. `address` must be `host:port`.
- Kerberos creates the tables. Start Kerberos against the database once before you start the connector.

See [Persistence](./persistence.md) for all keys and [Sharing the database with admin-connector](./persistence.md#sharing-the-database-with-admin-connector).

### Origins

`origins` uses the same format and modes as everywhere else in Kerberos (`allowedOrigins`, `allowAll` or `denyAll`); see [Origins and CORS](./cors.md). It applies to requests that carry an `Origin` header. Browsers send `Origin` on `POST` and similar requests from the tool's own pages, so for tools that make such requests, list the connector's own origin, for example:

```json
{
  "origins": {
    "allowedOrigins": ["https://ops.example.com:30100"]
  }
}
```

### TLS

| Key | Type | Description |
| --- | --- | --- |
| `tls.serverCertFile` | string | PEM certificate for the connector's port. Required when `tls` is set. |
| `tls.serverKeyFile` | string | PEM private key for that certificate. Required when `tls` is set. |

When both `tls` paths are set to non-empty values, the connector serves only HTTPS on `PORT`. Otherwise it serves plain HTTP. See [TLS](./tls.md) for how Kerberos loads certificates.

## Example: Jaeger behind the connector with Docker Compose

This is the setup `krbctl compose --obs-stack --connector` generates (see [krbctl](./krbctl.md)): the Jaeger UI is not published, and only reachable through the connector. Kerberos and the connector share the SQLite database through the `krbdata` volume.

`connector.json`:

```json
{
  "origins": {
    "denyAll": true
  },
  "persistence": {
    "driver": "sqlite",
    "address": "/krbdata/krb.db"
  }
}
```

`compose.yaml` (excerpt; the `kerberos` service uses the same `persistence` block and mounts `krbdata:/krbdata`):

```yaml
services:
  connector:
    image: "ghcr.io/trebent/kerberos/admin-connector:latest"
    command: --config /connector.json
    depends_on:
      kerberos:
        condition: service_started
    ports:
      - 30100:30100
    restart: on-failure
    environment:
      - LOG_TO_CONSOLE=true
      - PORT=30100
      - TARGET=jaeger:16686
      - OTEL_EXPORTER_OTLP_ENDPOINT=http://jaeger:4317
      - OTEL_EXPORTER_OTLP_TRACES_PROTOCOL=grpc
      - OTEL_METRICS_EXPORTER=prometheus
      - OTEL_EXPORTER_PROMETHEUS_HOST=connector
      - OTEL_EXPORTER_PROMETHEUS_PORT=9464
    volumes:
      - ./connector.json:/connector.json:ro
      - krbdata:/krbdata

  jaeger:
    image: "jaegertracing/jaeger:2.20.0"
    # No ports: the UI on 16686 is only reachable through the connector.

volumes:
  krbdata:
```

After `docker compose up -d`, log in on `http://localhost:30001` (for example with the superuser), then open `http://localhost:30100` in the same browser. Because everything runs on `localhost`, the `Secure` cookie is sent over plain HTTP.

## Metrics and traces

When `OBSERVABILITY_ENABLED` is `true`, the connector exports these counters:

| Metric | Counts |
| --- | --- |
| `admin_connector_calls_total` | Requests that passed the session check and were forwarded. |
| `admin_connector_calls_denied_total` | Requests rejected with `401`. |
| `admin_connector_callout_failures_total` | Forwarded requests that failed and were answered with `502`. |

For each request that reaches the session check, the connector starts a server span named `<METHOD> <URL>`, for example `GET /search?service=kerberos`, with the HTTP method and URL as attributes. The span continues an incoming trace when the request carries trace headers, and the trace context is passed on to the target. Denied requests end with an error status.

Each request is also logged with its method, path and status code.

## Things to know

- **One target per connector.** Run one connector per tool, each on its own port or host name.
- **Timeouts.** `WRITE_TIMEOUT_SECONDS` (see [Environment variables](#environment-variables)) also covers the time the target takes to answer. Raise it for tools with slow queries.
- **External URLs.** If a tool builds links from its own address, set the tool's external URL to the connector's address.

## Related docs

- [Sessions](./sessions.md): admin sessions, cookie attributes and refresh.
- [Admin API](./admin-api.md): admin users and the superuser.
- [Persistence](./persistence.md): database configuration shared with Kerberos.
- [Origins and CORS](./cors.md): the `origins` block.
- [TLS](./tls.md): certificates and HTTPS.
- [Observability](./observability.md): exporting metrics and traces.
- [krbctl](./krbctl.md): generating a Compose setup that includes the connector.
