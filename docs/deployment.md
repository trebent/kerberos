# Deployment

This page covers running Kerberos in a real environment: which images and binaries are published, how to start the container, the CLI flag and environment variables it reads, the two HTTP servers it runs, health checks, shutdown, timeouts and scaling. It ends with a checklist to work through before you put Kerberos in front of production traffic.

For a first local run, see [Getting started](./getting-started.md). For the configuration file itself, see [Configuration](./configuration.md).

## What ships

### Container images

Images are published to the GitHub Container Registry:

| Image | Contents | Listens on |
| --- | --- | --- |
| `ghcr.io/trebent/kerberos` | The Kerberos gateway, plus the OpenAPI files it needs at `/krb-oas` | `PORT` (gateway) and `ADMIN_PORT` (admin), see [Environment variables](#environment-variables) |
| `ghcr.io/trebent/kerberos/admin-connector` | Reverse proxy that puts internal tools behind an admin login. See [Admin connector](./admin-connector.md). | Its own `PORT`, see [Admin connector](./admin-connector.md#environment-variables) |
| `ghcr.io/trebent/kerberos/echo` | Small HTTP echo server, useful as a test backend | `15000` |

Every image is published with these tags:

| Tag | When it is published |
| --- | --- |
| `:<release>` (for example `:v1.2.0`) | When a GitHub release is published. The tag is the release's Git tag. |
| `:latest` | Together with every release tag. Points to the most recent release. |
| `:unstable` | On every push to the `main` branch. Not a release; use it only for testing. |

Pin a release tag in production rather than `:latest`, so that a new release does not reach you on the next image pull.

All images are based on a distroless, non-root base image (`gcr.io/distroless/static-debian12:nonroot`). There is no shell, `curl` or `wget` inside the container, and the process runs as the `nonroot` user. The image tag is also baked into the image as the `VERSION` environment variable, which Kerberos reports in its logs, traces and metrics.

### krbctl

`krbctl` is a command-line tool that generates a Docker Compose file and a starter Kerberos configuration. Each GitHub release has `krbctl` archives attached:

- `krbctl_<release>_<os>_<arch>.tar.gz` for `linux` and `darwin` (`amd64` and `arm64`)
- `krbctl_<release>_windows_<arch>.zip` for `windows` (`amd64` and `arm64`)
- `checksums.txt` with SHA-256 checksums of every archive

See [krbctl](./krbctl.md) for usage.

## Running the container

The image's entrypoint is the Kerberos binary, and it has no default command. You must pass `--config` with the path to a configuration file that you mount into the container:

```sh
docker run --rm \
  -p 30000:30000 \
  -p 30001:30001 \
  -v "$PWD/krb.json:/config/krb.json:ro" \
  -v krbdata:/krbdata \
  -e LOG_TO_CONSOLE=true \
  ghcr.io/trebent/kerberos:<release> \
  --config /config/krb.json
```

Things to know:

- **Mount the configuration read-only.** Kerberos only reads the file at startup. Changes require a restart.
- **`OAS_DIRECTORY` is already set** to `/krb-oas` in the image, which holds the OpenAPI files the admin and authentication APIs need. Do not override it unless you supply your own copy of the repository's `openapi/` directory.
- **Publish the admin port yourself.** The image only declares the default gateway port. The admin port and, if you use the Prometheus exporter, the metrics port (see [Observability](./observability.md#exporters-and-environment-variables)) must be published explicitly.
- **Give SQLite a writable location.** The default SQLite database path (see [Persistence](./persistence.md#key-reference)) is relative to the working directory, which is `/` in the container and not writable by the `nonroot` user. Mount a volume and set `persistence.address` to an absolute path on it, for example `/krbdata/krb.db`. The volume must be writable by the container user (UID 65532); a new Docker volume is owned by root, so prepare it first as shown in the [Compose example](#example-docker-compose). See [Persistence](./persistence.md).

## Command-line flags

| Flag | Required | Description |
| --- | --- | --- |
| `--config <path>` | Yes | Path to the JSON configuration file. If it is missing, Kerberos prints usage and exits with status 1. |
| `-h`, `--help` | No | Prints the flags and the list of environment variables with their defaults, then exits. |

There are no other flags. Everything else is set through the configuration file or environment variables.

## Environment variables

Kerberos reads the following variables at startup. Names have no prefix. A variable that is unset takes its default; a variable that is set is parsed and validated, and any failure stops Kerberos with exit status 1 and a message such as `variable validation failed: PORT: must be between 1000 and 65535: 80`, followed by the variable list.

| Variable | Type | Default | Validation | Description |
| --- | --- | --- | --- | --- |
| `PORT` | integer | `30000` | 1000–65535 | Port of the gateway server. |
| `ADMIN_PORT` | integer | `30001` | 1000–65535 | Port of the admin server. |
| `READ_TIMEOUT_SECONDS` | integer | `5` | greater than 0 | Maximum time to read a whole request (headers and body). Applies to both servers. |
| `WRITE_TIMEOUT_SECONDS` | integer | `5` | greater than 0 | Maximum time to produce and write a response. Applies to both servers. See [Timeouts](#timeouts). |
| `OAS_DIRECTORY` | string | empty (image: `/krb-oas`) | must be a readable directory | Directory containing `admin.yaml` and `auth_basic.yaml` from the repository's `openapi/` folder. Required in practice: if it is unset, startup fails with `failed to read admin OAS`. |
| `LOG_TO_CONSOLE` | boolean | `false` | `true`/`false`, `1`/`0`, `t`/`f` | `true` writes human-readable console logs; `false` writes one JSON object per line. |
| `LOG_VERBOSITY` | integer | `0` | 0 or greater | Log detail. `0` logs requests, startup and errors. Higher values (Kerberos uses levels up to `100`) add debug output. |
| `VERSION` | string | `unset` (image: the image tag) | none | Version reported in logs, traces (`service.version`) and metrics. |

Port values below 1000 are rejected, so Kerberos cannot listen on `80` or `443` directly. Put a load balancer or port mapping in front of it.

Keep `LOG_VERBOSITY` at `0` in production. Higher levels are meant for troubleshooting and log request details.

Kerberos also honours the standard OpenTelemetry environment variables (`OTEL_TRACES_EXPORTER`, `OTEL_METRICS_EXPORTER`, `OTEL_EXPORTER_OTLP_ENDPOINT`, `OTEL_EXPORTER_PROMETHEUS_HOST`, `OTEL_EXPORTER_PROMETHEUS_PORT`, `OTEL_TRACES_SAMPLER`, `OTEL_TRACES_SAMPLER_ARG` and others). They are described in [Observability](./observability.md). Two points matter for deployment:

- The Prometheus exporter binds to `localhost` by default (see [Observability](./observability.md#exporters-and-environment-variables)), which cannot be reached from outside a container. Set `OTEL_EXPORTER_PROMETHEUS_HOST=0.0.0.0` (or the container's host name) and publish the port.
- If observability is enabled (the default) and you set no exporter variables, Kerberos tries to export over OTLP to a local collector. Either point it at a collector, set the exporters to `none`, or set `observability.enabled` to `false` in the configuration.

Configuration values can also come from the environment through `${env:NAME}` references in the configuration file. See [Configuration](./configuration.md).

## The two servers

Kerberos runs two HTTP servers in one process. Both start only after the configuration has been validated, the database has been reached and its tables created, and the OpenAPI files have been loaded. If any of that fails, the process exits with status 1 before listening.

| Server | Port | Paths served | TLS setting |
| --- | --- | --- | --- |
| Gateway | `PORT` | `/gw/backend/<name>/...` (proxied requests, see [Routing](./routing.md)) and `/gw/health`. Everything outside `/gw/` returns a plain-text `404 page not found`. | `gateway.tls` |
| Admin | `ADMIN_PORT` | `/api/admin/...` (see [Admin API](./admin-api.md)) and, when basic authentication is configured, `/api/auth/basic/...` (see [Organisations](./organizations.md)). | `admin.api.tls` |

The admin server always runs, even if the configuration has no `admin` section. It is what operators and your login pages talk to, so decide deliberately who can reach it. A common setup exposes the gateway port publicly and keeps the admin port on an internal network, unless browsers need the basic-auth login API, which is also served on the admin port.

Enabling TLS on a server makes that port HTTPS only. See [TLS](./tls.md).

## Health checks and readiness

The gateway server answers `/gw/health` with `204 No Content` for any HTTP method. The endpoint does no further checks: it does not test the database or the backends, and it is not subject to authentication, request validation or CORS.

```sh
curl -i http://localhost:30000/gw/health
# HTTP/1.1 204 No Content
```

Because the servers only start listening after initialisation has completed, a `204` from `/gw/health` also means Kerberos is ready to serve traffic. Use it as both the liveness and readiness probe.

- Kubernetes: use an `httpGet` probe on `/gw/health` (with `scheme: HTTPS` if `gateway.tls` is set).
- Docker / Docker Compose: an in-container `healthcheck` is not possible because the image has no shell or HTTP client. Probe from outside the container instead.

The admin server has no health endpoint. If you need to know that it is up, send an unauthenticated `GET /api/admin/flow` and expect `401`.

## Graceful shutdown

On `SIGINT` or `SIGTERM`, Kerberos stops accepting new connections on both servers and waits for in-flight requests to finish. The wait is limited to `READ_TIMEOUT_SECONDS + WRITE_TIMEOUT_SECONDS`, shared by both servers. When shutdown completes it logs `Kerberos stopped` and exits with status 0.

Kerberos stops listening as soon as it receives the signal; `/gw/health` does not first switch to failing. In Kubernetes, if you want load balancers to drain the pod before it stops accepting connections, add a short `preStop` sleep. Keep the pod's termination grace period longer than the preStop sleep plus the shutdown wait.

## Timeouts

Three settings limit how long a request can take:

| Setting | Where | Limits |
| --- | --- | --- |
| `READ_TIMEOUT_SECONDS` | Environment, see [Environment variables](#environment-variables) | Time for the client to send the full request, including the body. |
| `WRITE_TIMEOUT_SECONDS` | Environment, see [Environment variables](#environment-variables) | Time from the end of reading the request headers until the response has been written. This includes the whole round trip to the backend. |
| `timeout` per backend | `gateway.router.backends[]`, see [Routing: backend keys](./routing.md#backend-keys) | Time for the complete exchange with that backend, including reading its response body. |

The server write timeout covers the backend call, so it is the effective upper limit. If a backend takes longer than `WRITE_TIMEOUT_SECONDS`, Kerberos still waits for it (up to the backend `timeout`), but the client's connection is then closed without any response, even if the backend answered successfully. If the backend `timeout` is reached first, the client gets `500` with `{"errors":["failed to forward request"]}`.

For slow backends, raise both: set the backend `timeout` to what the backend needs, and set `WRITE_TIMEOUT_SECONDS` a few seconds above the largest backend timeout. Raise `READ_TIMEOUT_SECONDS` if clients upload large bodies over slow links. Remember that both values also set the shutdown wait, and both apply to the admin server as well.

## Example: Docker Compose

A minimal setup with the echo test backend and SQLite on a named volume. The `sqlite-init` service prepares the volume so that the non-root Kerberos user can write to it.

`krb.json`:

```json
{
  "gateway": {
    "router": {
      "backends": [
        { "name": "echo", "host": "echo", "port": 15000 }
      ]
    }
  },
  "admin": {
    "superUser": {
      "clientId": "admin",
      "clientSecret": "${env:KRB_SUPERUSER_SECRET}"
    }
  },
  "persistence": {
    "driver": "sqlite",
    "address": "/krbdata/krb.db"
  },
  "observability": {
    "enabled": false
  }
}
```

`compose.yaml`:

```yaml
services:
  sqlite-init:
    image: busybox:1.38
    command: ["sh", "-c", "chmod 0777 /krbdata && touch /krbdata/krb.db && chmod 0666 /krbdata/krb.db"]
    volumes:
      - krbdata:/krbdata

  kerberos:
    image: ghcr.io/trebent/kerberos:<release>
    command: --config /config/krb.json
    depends_on:
      sqlite-init:
        condition: service_completed_successfully
    restart: on-failure
    ports:
      - "30000:30000"
      - "30001:30001"
    environment:
      - LOG_TO_CONSOLE=true
      - KRB_SUPERUSER_SECRET=change-me
    volumes:
      - ./krb.json:/config/krb.json:ro
      - krbdata:/krbdata

  echo:
    image: ghcr.io/trebent/kerberos/echo:<release>
    environment:
      - PORT=15000

volumes:
  krbdata:
```

The echo service publishes no ports, so it is only reachable through Kerberos:

```sh
docker compose up -d
curl -i http://localhost:30000/gw/health
curl -i http://localhost:30000/gw/backend/echo/hello
```

`krbctl compose` and `krbctl config` generate a fuller version of this, optionally with PostgreSQL, Prometheus, Grafana and Jaeger. The repository's `test/compose/integration/compose.yaml` shows a PostgreSQL and full observability setup.

If you use PostgreSQL, Kerberos exits at startup when it cannot reach the database; it does not retry. Use `depends_on` with a database health check, and a restart policy.

## Scaling

- **SQLite means one instance.** Kerberos opens a single connection to the SQLite file, and the file cannot be shared safely between processes or hosts. Run exactly one Kerberos replica with SQLite.
- **Use PostgreSQL for more than one replica.** All persistent state (admin users and groups, the superuser, organisations, users, groups, login sessions and recorded debug calls) lives in the database, so replicas sharing a PostgreSQL database share logins. See [Persistence](./persistence.md).
- **Debug sessions are per instance.** Which backends are currently being debugged is held in memory by the instance that received the admin API call. Other replicas do not record calls for that session, and the state is lost on restart. With several replicas, a debug session only captures the share of traffic that lands on that one instance. See [Admin debugging](./admin-debugging.md).
- **The configuration must be identical on every replica.** Each instance reads its own file at startup.

## Production checklist

- [ ] **Change the superuser secret.** The default superuser credentials are public (see [Admin API: the superuser](./admin-api.md#the-superuser)). The configured credentials are only used to create the superuser on the first start against an empty database; changing the configuration later has no effect. Set a strong `admin.superUser.clientSecret` before the first start, and change the password through the admin API (`PUT /api/admin/superuser/password`) afterwards. See [Admin API](./admin-api.md).
- [ ] **Enable TLS** on the gateway and admin servers, or terminate TLS in front of them. Kerberos's session cookies are always marked `Secure`, so browser logins only work over HTTPS. See [TLS](./tls.md).
- [ ] **Make backends reachable only through Kerberos.** Keep backends on a private network so that every request passes through Kerberos's checks, and rely on the `X-Krb-*` identity headers only on routes that Kerberos authenticates. See [Authentication](./authentication.md).
- [ ] **Restrict access to the admin port** to the networks that need it.
- [ ] **Use PostgreSQL** if you run more than one replica, and back it up. With SQLite, keep the database on a persistent volume and run a single replica. See [Persistence](./persistence.md).
- [ ] **Configure telemetry deliberately.** Point the OTLP exporter at a collector or set exporters to `none`, expose the Prometheus port if you scrape metrics, and set a trace sampler such as `OTEL_TRACES_SAMPLER=parentbased_traceidratio` with `OTEL_TRACES_SAMPLER_ARG=0.1`. The default samples every trace. See [Observability](./observability.md).
- [ ] **Set timeouts** so that `WRITE_TIMEOUT_SECONDS` is larger than your largest backend `timeout`.
- [ ] **Keep `LOG_VERBOSITY` at `0`** unless you are troubleshooting.
- [ ] **Pin an image release tag** instead of `:latest` or `:unstable`.
- [ ] **Keep secrets out of the configuration file** by using `${env:NAME}` references. See [Configuration](./configuration.md).

## Related docs

- [Configuration](./configuration.md)
- [TLS](./tls.md)
- [Persistence](./persistence.md)
- [Observability](./observability.md)
- [Authentication](./authentication.md)
- [Admin API](./admin-api.md)
- [Routing](./routing.md)
- [krbctl](./krbctl.md)
