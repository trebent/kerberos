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
  - Per-backend TLS configuration
  - Per-backend CORS configuration
  - Per-backend timeout management
- Per-backend forwarding with host and port configuration
- OpenTelemetry-based tracing and request/response metrics
- JSON-based configuration with environment and in-file reference resolution
- Schema-validated configuration loading at startup
- OpenAPI request validation per backend
- Authentication and authorization middleware in the gateway flow
- Basic authentication with session management via `X-Krb-Session`
  - Multi-tenant organisation model with users, groups, and group-based authorization
- Organisation administrators and super-user support for management operations
- Persistence configuration for SQLite and PostgreSQL backends

## Documentation

Detailed documentation lives in [`/docs`](./docs). Start with [`/docs/README.md`](./docs/README.md) for the documentation index and entry point.
