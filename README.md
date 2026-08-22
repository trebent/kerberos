# Kerberos

[![Main branch protection](https://github.com/trebent/kerberos/actions/workflows/main.yaml/badge.svg)](https://github.com/trebent/kerberos/actions/workflows/main.yaml)
[![Code scanning](https://github.com/trebent/kerberos/actions/workflows/code-scanning.yaml/badge.svg)](https://github.com/trebent/kerberos/actions/workflows/code-scanning.yaml)
[![License](https://img.shields.io/badge/License-BSD_3--Clause-blue.svg)](https://opensource.org/licenses/BSD-3-Clause)
[![Go Report Card](https://goreportcard.com/badge/github.com/trebent/kerberos)](https://goreportcard.com/report/github.com/trebent/kerberos)
![tag](https://img.shields.io/github/v/tag/trebent/kerberos?label=latest%20version)

🐶🐶🐶

Kerberos is an API gateway for routing backend traffic with built-in observability and request controls. It is designed around a fixed request flow (Observability → Router → Custom components → Forwarder), where custom components can enforce security and contract validation before forwarding traffic to backend services.

## Currently supported features

- Backend routing through `/gw/backend/<backend-name>/<backend-path>`
  - Per-backend forwarding with host and port configuration
  - Per-backend TLS configuration
  - Per-backend CORS configuration
  - Per-backend timeout management
- OpenTelemetry-based tracing and request/response metrics
- Request-flow debugging through admin debug sessions and captured flow transitions
- OpenAPI request validation per backend
- Authentication and authorization middleware in the gateway flow
- Basic authentication with cookie-based session management
  - Multi-tenant organisation model for backend-access users, groups, and group-based authorization
  - Organisation-level administrator accounts for tenant user and group management
- Admin API user model for platform administration and super-user access
- Persistence backends: SQLite and PostgreSQL

## Documentation

Detailed documentation lives in [`/docs`](./docs).

- Start with [`/docs/README.md`](./docs/README.md) for the documentation index.
- **User docs** explain how to deploy, configure, and operate Kerberos.
- **Developer docs** (under [`/docs/dev`](./docs/dev)) cover contributor-facing internals and implementation details.
