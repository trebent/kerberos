# Developer documentation

These pages are for people who change Kerberos itself. You do not need them to deploy, configure or integrate with Kerberos; for that, see the [user documentation](../README.md).

## Before you write code

Read the design rules in [`.github/skills/design/SKILL.md`](../../.github/skills/design/SKILL.md). They apply to all Go code in this repository and cover DRY/KISS, Opts structs, generics, interfaces, error handling, constructors and context propagation. Then read the user docs for the area you are changing. [Routing: request lifecycle](../routing.md#request-lifecycle) is the best overview of the system.

## Internals

- [Flow components](./flow-components.md): the `FlowComponent` interface, the fixed request flow (Observability → Router → Custom → Forwarder), ordered custom-block components and the request-context guarantees.
- [Middleware](./middleware.md): the order in which oapi-codegen standard and strict middleware run.

## Repository layout

| Path | Contents |
| --- | --- |
| `main.go`, `env.go` | The Kerberos gateway entry point and its environment variables |
| `cmd/` | Other binaries: `admin-connector`, `echo` (test backend) and `krbctl` |
| `internal/composer/` | The request flow: observability, router, custom block and forwarder |
| `internal/auth/`, `internal/oas/` | The `authorizer` (with its auth methods, including basic auth and its API) and the OpenAPI validator |
| `internal/admin/` | The Admin API, including debug sessions |
| `internal/config/` | Config loading, reference resolution and the JSON schemas (`schemas/`) |
| `internal/db/` | SQLite and PostgreSQL persistence |
| `pkg/` | Public Go packages: `pkg/auth` (identity middleware and basic-auth client) and `pkg/admin` (Admin API client) |
| `openapi/` | OpenAPI contracts for the gateway, Admin API and basic-auth API; code is generated from them |
| `test/` | Integration, security, connector and krbctl suites, plus their configs, certificates and Compose files. See [`test/README.md`](../../test/README.md). |
| `docs/` | This documentation |
| `pages/` | The GitHub Pages site: an Angular app that renders the API reference from `openapi/`. See [`pages/README.md`](../../pages/README.md). |

## Build and test

| Command | What it does |
| --- | --- |
| `make install/deps` | Installs code-generation and tooling dependencies |
| `make codegen` | Regenerates code from the OpenAPI contracts |
| `make build` | Builds the Kerberos binary into `build/` |
| `make run` | Runs Kerberos locally with `test/config/local.json` (start `make echo/run` first) |
| `make validate` | Lint, unit tests and vulnerability check: run this before opening a pull request |
| `make test/unit` | Unit tests only |
| `make pages/install`, then `make pages/serve` or `make pages/check` | Runs the GitHub Pages site locally, or runs its CI checks |
| `make compose/up`, then `make test/integration` | Starts the integration environment and runs the integration suite |

The `security` and `connector` suites follow the same pattern (`make compose/<suite>/up`, then `make test/<suite>`). `make test/krbctl` builds `krbctl` and tests it without a Compose environment. `make build` produces a Linux binary (`GOOS=linux`). The CI workflows in `.github/workflows/` run these targets.

## Task guides

`.github/skills/` holds step-by-step guides for common changes, and `.github/agents/` holds multi-step agent definitions. They are written for AI coding agents, but people can follow them too:

| Guide | Use it when you are |
| --- | --- |
| [`configuration`](../../.github/skills/configuration/SKILL.md) | Adding or changing a config section or schema |
| [`flow-components`](../../.github/skills/flow-components/SKILL.md) | Implementing or debugging a flow component |
| [`oas`](../../.github/skills/oas/SKILL.md) | Adding or changing an OpenAPI specification |
| [`validation`](../../.github/skills/validation/SKILL.md) | Checking a change end to end before merging |

## Keeping the docs current

If your change alters behaviour that a user can see, such as a config key, a status code, a header or a default, update the matching page under `docs/` in the same pull request. Each setting is documented in one place, so a change usually needs one doc edit:

| What changed | The page to update |
| --- | --- |
| A top-level config key | [`configuration.md`](../configuration.md#root-keys), and add the new section's page |
| A key inside a config section or shared object | The page listed for it in [Configuration: root keys](../configuration.md#root-keys) |
| A Kerberos environment variable | [`deployment.md`](../deployment.md#environment-variables) |
| An admin-connector config key or environment variable | [`admin-connector.md`](../admin-connector.md) |
| An echo environment variable | [`echo.md`](../echo.md#configuration) |

Other pages link to that page rather than repeat a key's type, allowed values or default. Add any new page to the index in [`docs/README.md`](../README.md#pages).
