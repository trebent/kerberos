# Kerberos documentation

Kerberos is an API gateway that puts your backend services behind a single entry point. Clients call `/gw/backend/<name>/...` on the gateway port (`30000`), and Kerberos forwards the request to the backend registered under that name. Per backend, you can add login-based access control, validate requests against an OpenAPI specification, and control which browser origins may call it, with TLS available on every hop. Kerberos also emits metrics, traces and logs, and serves a platform Admin API and the basic-auth login API on a separate admin port (`30001`), where you can also open debug sessions to see how requests move through the gateway.

## How a request travels

```text
                          Gateway port (30000)
client --> /gw/backend/<name>/<path>
             |
             +- router: find backend <name> (404 if unknown)
             |          check Origin / CORS (403 if not allowed)
             |
             +- access control (auth)        when configured, in the order
             +- request validation (oas)     set by each section's "order"
             |
             `- forward <path> --> backend service

                          Admin port (30001)
operators, login pages --> /api/admin/...       Admin API
                       --> /api/auth/basic/...  basic-auth API (organisation logins)
```

## Start here

New to Kerberos? Follow [Getting started](./getting-started.md) to run it locally and send your first request through it. When you are ready to run it for real, read [Deployment](./deployment.md).

| You are… | Read these first |
| --- | --- |
| **Evaluating Kerberos** | The [project README](../README.md) for the feature overview, then [Getting started](./getting-started.md). The *Limitations* and *Things to know* sections at the end of each page list what is not supported yet. |
| **Deploying and operating it** | [Deployment](./deployment.md) (with its production checklist), [Configuration](./configuration.md), [Persistence](./persistence.md), [TLS](./tls.md), [Observability](./observability.md) |
| **Putting your service behind it** | [Routing](./routing.md), [Authentication](./authentication.md) (identity headers and the Go middleware), [Request validation](./request-validation.md), [CORS](./cors.md) |
| **Building a login page or client** | [Sessions](./sessions.md), [Organisations](./organizations.md), [CORS](./cors.md), and the [API reference](#api-reference) |
| **Contributing code** | [Developer documentation](./dev/README.md) |

## Getting started and operations

- [Getting started](./getting-started.md): a first-run tutorial. The recommended path uses krbctl to generate a Docker Compose setup with a protected echo test backend; the alternative runs Kerberos from source. Both end with a logged-in user calling a protected backend with curl.
- [Deployment](./deployment.md): everything needed to run Kerberos outside a laptop: images and tags, the `--config` flag, environment variables, the two servers, health checks, shutdown and timeouts. Ends with scaling limits and a production checklist.
- [Configuration](./configuration.md): the JSON config file, how it is loaded and validated, and the `${env:...}` and `${ref:...}` references. Includes an overview of every section, the effective defaults and a complete annotated example.
- [Persistence](./persistence.md): the database that holds admin users, sessions, organisations and debug captures. Explains choosing between SQLite and PostgreSQL, configuring it, automatic table creation, and backups.

## Routing traffic

- [Routing](./routing.md): how to register backends and reach them at `/gw/backend/<name>/<path>`. Covers naming rules, path stripping, what is and is not forwarded, timeouts, the health endpoint and every error status.
- [TLS](./tls.md): serving HTTPS on the gateway and admin ports, and using HTTPS or mutual TLS towards backends. Also explains why login cookies need HTTPS and how to run behind a TLS-terminating proxy.
- [Origins and CORS](./cors.md): controlling which browser origins may call gateway backends, the Admin API and the basic-auth API. Use it when browser applications on other origins call Kerberos.
- [Request validation](./request-validation.md): checking requests against a backend's OpenAPI 3.0 specification before forwarding them. Unknown paths, bad parameters and invalid JSON bodies are rejected at the gateway, so the backend only sees well-formed requests.

## Users and access control

- [Authentication and authorization](./authentication.md): protecting gateway backends so that only logged-in organisation users, optionally limited by group, can reach them. Kerberos passes the caller's identity to the backend in `X-Krb-*` headers, and Go backends can read it with the `pkg/auth` middleware.
- [Sessions and cookies](./sessions.md): how login, refresh and logout work for both the basic-auth API and the Admin API. Covers the session, refresh and CSRF cookies, their lifetimes, CSRF protection and cookie settings.
- [Organisations, users and groups](./organizations.md): the multi-tenant model behind basic authentication and the basic-auth API that manages it. Explains who can do what, with a walkthrough from creating an organisation to a user's first login.
- [Admin API](./admin-api.md): the platform administration interface on the admin port. Covers the superuser, admin users and groups, permissions, inspecting the request flow and OpenAPI specifications, and the Go client.

## Operating and troubleshooting

- [Observability](./observability.md): the metrics, traces and logs Kerberos produces and how to export them with standard OpenTelemetry environment variables. Includes a metrics reference, log verbosity levels and example dashboards and queries.
- [Debug sessions](./admin-debugging.md): recording how individual requests to one backend move through the gateway, step by step, and why they failed. Use it to troubleshoot a specific backend without turning on tracing.

## Tools

- [krbctl](./krbctl.md): a command-line tool that scaffolds a Kerberos deployment. It writes a Docker Compose file and a matching config, optionally with PostgreSQL, the echo test backend, the admin connector and an observability stack.
- [Admin connector](./admin-connector.md): a separate reverse proxy that puts internal tools, such as the Jaeger UI, behind Admin API login. Only requests with a valid admin or superuser session get through.

## API reference

The HTTP APIs are described by OpenAPI contracts in [`../openapi/`](../openapi/):

- [`gateway.yaml`](../openapi/gateway.yaml): the gateway port.
- [`admin.yaml`](../openapi/admin.yaml): the Admin API.
- [`auth_basic.yaml`](../openapi/auth_basic.yaml): the basic-auth API.

A [rendered reference](https://trebent.github.io/kerberos/) is published with GitHub Pages from [`pages/index.html`](../pages/index.html).

## Developer documentation

[Developer documentation](./dev/README.md) covers internal architecture and implementation details. It is for contributors to Kerberos, not for operating or integrating it.
