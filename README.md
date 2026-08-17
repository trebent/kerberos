# Kerberos

[![Main branch protection](https://github.com/trebent/kerberos/actions/workflows/main.yaml/badge.svg)](https://github.com/trebent/kerberos/actions/workflows/main.yaml)
[![Code scanning](https://github.com/trebent/kerberos/actions/workflows/code-scanning.yaml/badge.svg)](https://github.com/trebent/kerberos/actions/workflows/code-scanning.yaml)
[![License](https://img.shields.io/badge/License-BSD_3--Clause-blue.svg)](https://opensource.org/licenses/BSD-3-Clause)
[![Go Report Card](https://goreportcard.com/badge/github.com/trebent/kerberos)](https://goreportcard.com/report/github.com/trebent/kerberos)
![tag](https://img.shields.io/github/v/tag/trebent/kerberos?label=latest%20version)

🐶🐶🐶

Kerberos is a configurable API gateway for routing backend traffic with built-in observability and optional request controls. It is designed around a fixed request flow (Observability → Router → Custom components → Forwarder), where optional custom components can enforce security and contract validation before forwarding traffic to backend services.

## Currently supported features

- Backend routing through `/gw/backend/<backend-name>/<backend-path>`
- Per-backend forwarding with configurable host, port, timeout, and TLS settings
- OpenTelemetry-based tracing and request/response metrics
- JSON-based configuration with environment and in-file reference resolution
- Schema-validated configuration loading at startup
- Optional OpenAPI request validation per backend
- Optional authentication and authorization middleware in the gateway flow
- Basic authentication with session management via `X-Krb-Session`
- Multi-tenant organisation model with users, groups, and group-based authorization
- Organisation administrators and super-user support for management operations

## Documentation

Detailed documentation lives in [`/docs`](./docs). Start with [`/docs/README.md`](./docs/README.md) for the documentation index and entry point.
