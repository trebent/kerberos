# Kerberos

[![Main branch protection](https://github.com/trebent/kerberos/actions/workflows/main.yaml/badge.svg)](https://github.com/trebent/kerberos/actions/workflows/main.yaml)
[![Code scanning](https://github.com/trebent/kerberos/actions/workflows/code-scanning.yaml/badge.svg)](https://github.com/trebent/kerberos/actions/workflows/code-scanning.yaml)
[![License](https://img.shields.io/badge/License-BSD_3--Clause-blue.svg)](https://opensource.org/licenses/BSD-3-Clause)
[![Go Report Card](https://goreportcard.com/badge/github.com/trebent/kerberos)](https://goreportcard.com/report/github.com/trebent/kerberos)
![tag](https://img.shields.io/github/v/tag/trebent/kerberos?label=latest%20version)

🐶🐶🐶 **A self-hosted API gateway with login, multi-tenant access control, OpenAPI enforcement and observability built in.**

Kerberos is named after the three-headed dog that guards the gate. It puts your backend services behind one entry point and checks every request before it reaches them. It does three jobs: **routing** traffic, **guarding** backends, and **observing** what happens to each request. You do not have to assemble plugins, sidecars or a separate identity service to get there.

> 📚 **Documentation:** start at **[`docs/`](./docs/README.md)**. The [Getting started](./docs/getting-started.md) tutorial goes from nothing to a logged-in user calling a protected backend in about five minutes.

## Why Kerberos

- **Login and access control included.** Organisations, users and groups are managed through an API. Users log in with cookie sessions protected against CSRF, and per-backend rules decide who gets through. Your backend receives the caller's identity in `X-Krb-*` headers, and Go services can read them with one middleware call.
- **Your OpenAPI contract, enforced at the edge.** Map a backend to its OpenAPI 3 spec, and requests with unknown paths, missing parameters or invalid bodies are rejected before they reach your code.
- **Observability you don't have to wire up.** OpenTelemetry traces and metrics come out of the box, configured with the standard `OTEL_*` variables. `krbctl` can generate Prometheus, Grafana dashboards and Jaeger for you.
- **See why a request failed.** Start a debug session for one backend and see each request step by step: which stage accepted or rejected it, how long each stage took, and why. You don't have to turn on tracing for the whole gateway.
- **Secure defaults.** TLS on every hop, mutual TLS to backends, per-backend CORS, `Secure`/`HttpOnly` cookies and bcrypt-hashed passwords. The images are minimal and run as a non-root user.
- **Small footprint.** One static Go binary and one JSON config file. Start with an embedded SQLite file and move to PostgreSQL when you run more than one replica.
- **Built for operators.** An Admin API with fine-grained permissions, a scaffolding CLI (`krbctl`), and an [admin connector](./docs/admin-connector.md) that puts internal tools such as the Jaeger UI behind the same admin login.

## How it works

```text
client ──▶ :30000 /gw/backend/<name>/<path>
             │
             ├─ observe   traces, metrics, logs, debug-session capture
             ├─ route     find backend <name>, check CORS origin
             ├─ guard     login + group rules, OpenAPI validation (each optional, per backend)
             └─ forward   <path> ──▶ your backend

operators, login pages ──▶ :30001  Admin API · organisation login API · debug sessions
```

Clients call `/gw/backend/<name>/<path>` on the gateway port. Kerberos finds the backend, runs it through the checks you configured, and forwards `<path>` to it. Logins, administration and debugging live on a separate admin port, so you can keep that port private.

## Quick start

With Docker and [`krbctl`](./docs/krbctl.md) (download it from the [releases page](https://github.com/trebent/kerberos/releases), or run `go install github.com/trebent/kerberos/cmd/krbctl@latest`):

```bash
mkdir kerberos-demo && cd kerberos-demo
krbctl compose -y --echo          # compose.yaml: Kerberos + an echo test backend + SQLite
krbctl config  -y --echo-backend  # krb.json: route /gw/backend/echo/... to the echo service
docker compose up -d

curl -i http://localhost:30000/gw/backend/echo/hello
```

The [echo backend](./docs/echo.md) answers with exactly what Kerberos forwarded. Next, follow [Getting started](./docs/getting-started.md) to protect it with a login, or add `--obs-stack` to both commands to get Grafana dashboards and Jaeger traces.

> The first start creates a superuser with [default credentials](./docs/admin-api.md#the-superuser). Change them before you expose Kerberos. See the [production checklist](./docs/deployment.md#production-checklist).

## Features

| Area | What you get | Docs |
| --- | --- | --- |
| Routing | Named backends at `/gw/backend/<name>/...`, per-backend timeouts, HTTPS and mTLS to backends, health endpoint | [Routing](./docs/routing.md), [TLS](./docs/tls.md) |
| Access control | Basic-auth logins, multi-tenant organisations, users and groups, per-path group rules, exempt paths, identity headers for backends | [Authentication](./docs/authentication.md), [Organisations](./docs/organizations.md), [Sessions](./docs/sessions.md) |
| Contract validation | Request validation against each backend's OpenAPI 3.0 spec, with or without body checks | [Request validation](./docs/request-validation.md) |
| Browser support | Per-backend and per-API CORS, CSRF double-submit protection, configurable cookie `Domain` and `SameSite` | [CORS](./docs/cors.md), [Sessions](./docs/sessions.md) |
| Observability | OpenTelemetry traces and metrics, Prometheus or OTLP export, structured logs, ready-made Grafana dashboards | [Observability](./docs/observability.md) |
| Troubleshooting | Debug sessions that record each request's path through the gateway | [Debug sessions](./docs/admin-debugging.md) |
| Administration | Admin API with a superuser, admin users, groups and seven fine-grained permissions; flow inspection; generated Go clients | [Admin API](./docs/admin-api.md) |
| Storage | SQLite for a single instance, PostgreSQL for several replicas | [Persistence](./docs/persistence.md) |
| Tooling | `krbctl` to scaffold Compose deployments; admin connector to protect internal tools | [krbctl](./docs/krbctl.md), [Admin connector](./docs/admin-connector.md) |

## Documentation

All documentation lives in **[`docs/`](./docs/README.md)**, which lists every page and what it is for. The HTTP API contracts are in [`openapi/`](./openapi).

## Contributing

Issues and pull requests are welcome. Start with the [developer documentation](./docs/dev/README.md), which covers the repository layout, the build and test commands, and the design rules that all Go code in this repository follows.

## Licence

Kerberos is released under the [BSD 3-Clause License](./LICENSE).
