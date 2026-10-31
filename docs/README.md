# Kerberos documentation

Kerberos is an API gateway that puts your backend services behind one entry point. Clients call `/gw/backend/<name>/...` on the gateway port, and Kerberos checks each request and forwards it to the backend with that name. Logins, administration and debugging use a separate admin port.

New to Kerberos? Start with [Getting started](./getting-started.md).

## Pages

| Page | Go here to… |
| --- | --- |
| [Getting started](./getting-started.md) | Run Kerberos on your machine and send your first request through it |
| [Echo test backend](./echo.md) | Use the echo service to see what Kerberos sends to a backend |
| [krbctl](./krbctl.md) | Install `krbctl` and generate a Docker Compose setup and config |
| [Deployment](./deployment.md) | Run Kerberos for real: images, environment variables, health checks, timeouts, scaling and the production checklist |
| [Configuration](./configuration.md) | Write the config file, use environment values in it, and find the page for each section |
| [Persistence](./persistence.md) | Choose and set up SQLite or PostgreSQL, and back up your data |
| [Routing](./routing.md) | Add backends and understand what Kerberos forwards to them |
| [TLS](./tls.md) | Serve HTTPS, and use HTTPS or mutual TLS towards backends |
| [Origins and CORS](./cors.md) | Let browser apps on other sites call Kerberos |
| [Request validation](./request-validation.md) | Check requests against a backend's OpenAPI specification |
| [Authentication and authorisation](./authentication.md) | Require a login for a backend, limit it to groups, and read the caller's identity in your service |
| [Sessions and cookies](./sessions.md) | Log in, refresh and log out, and set up cookies and CSRF protection |
| [Organisations, users and groups](./organizations.md) | Manage the organisations, users and groups that log in to your backends |
| [Admin API](./admin-api.md) | Manage the superuser, admin users, groups and permissions |
| [Debug sessions](./admin-debugging.md) | See step by step why requests to one backend fail |
| [Observability](./observability.md) | Collect metrics, traces and logs |
| [Admin connector](./admin-connector.md) | Put internal tools, such as the Jaeger UI, behind the admin login |
| [Developer documentation](./dev/README.md) | Change Kerberos itself |

## Conventions

- **One page per setting.** Each config section, shared config object and environment variable is described on one page, and other pages link there. [Configuration: root keys](./configuration.md#root-keys) lists the page for each config section, and [Deployment: environment variables](./deployment.md#environment-variables) lists the environment variables.
- **Examples use the default ports**: `30000` for the gateway and `30001` for the admin server, as set by `PORT` and `ADMIN_PORT` (see [Deployment: environment variables](./deployment.md#environment-variables)). If you change them, change the examples to match.

## API reference

The HTTP APIs are described by OpenAPI contracts in [`../openapi/`](../openapi/). The [API reference site](https://trebent.github.io/kerberos/) renders each one, with a view of its YAML source:

| API | Rendered reference | Contract | Guide |
| --- | --- | --- | --- |
| Gateway port | [Gateway API](https://trebent.github.io/kerberos/api/gateway) | [`gateway.yaml`](../openapi/gateway.yaml) | [Routing](./routing.md) |
| Admin API | [Admin API](https://trebent.github.io/kerberos/api/admin) | [`admin.yaml`](../openapi/admin.yaml) | [Admin API](./admin-api.md) |
| Basic-auth API | [Basic-auth API](https://trebent.github.io/kerberos/api/auth) | [`auth_basic.yaml`](../openapi/auth_basic.yaml) | [Organisations](./organizations.md), [Sessions](./sessions.md) |

The site's source is in [`../pages/`](../pages/README.md).
