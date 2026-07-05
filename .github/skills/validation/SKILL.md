---
name: validation
description: 'Validate a Kerberos PR end-to-end: lint, vulncheck, unit tests, integration tests, security tests, OAS endpoint coverage, and FlowComponent black-box coverage. Everything a PR needs to pass CI and merge to main.'
argument-hint: 'Optional: specific validation stage to run (lint, vulncheck, unittest, integrationtest, securitytest, coverage)'
---

# Validation — Kerberos

## When to Use

- Verifying that a code change is ready to merge (all CI checks pass locally)
- Running a specific validation stage after a targeted change
- Checking OAS endpoint coverage before opening a PR
- Verifying FlowComponent black-box test coverage after adding or modifying a component
- Debugging CI failures by reproducing them locally

---

## Validation Stages

| Stage | Make target(s) | What it checks |
|---|---|---|
| Lint | `make static-analysis/lint` | golangci-lint on main module |
| Vulnerability scan | `make static-analysis/vulncheck` | govulncheck on main module |
| Codegen drift | `make codegen` + git diff | Generated code is up-to-date with OAS specs |
| Build | `make build` | Main binary compiles |
| Unit tests | `make test/unit` | All `*_test.go` files in the main module |
| Postgres DB backend tests | `make postgres/run && make test/unit/postgres && make postgres/stop` | All `*_test.go` files tagged with 'postgres_integration' |
| Integration tests | `make compose/up && && make test/integration && make compose/down` | All tests in `test/suites/integration/` against a live environment |
| Security tests | `make compose/security/up && make test/security && make compose/security/down` | All tests in `test/suites/security/` against a TLS-enabled environment |

---

## Full Validation Workflow

Run these steps in order to reproduce a complete CI pass locally:

```sh
# 1. Lint and static analysis
make static-analysis/lint
make static-analysis/vulncheck

# 2. Verify generated code is current
make codegen
git diff --exit-code

# 3. Build
make build

# 4. Unit tests
make test/unit
make postgres/run
make test/unit/postgres
make postgres/stop

# 5. Integration tests
make compose/up
make test/integration
make compose/down

# 6. Security tests
make compose/security/up
make test/security
make compose/security/down
```

If any step fails, fix it before proceeding. All stages must pass for a PR to merge.

## Stage Reference

### Lint

```sh
make static-analysis/lint
```

Runs `golangci-lint run --fix` on the main module. Configuration is in `.golangci.yaml`.
Fix all reported issues before committing.

### Vulnerability scan

```sh
make static-analysis/vulncheck
```

Runs `govulncheck ./...` using the toolchain from `tools/go.mod`. Vulnerabilities in
transitive dependencies must be resolved by updating the affected module.

### Codegen drift

```sh
make codegen
git diff --exit-code
```

Regenerates all server and client boilerplate from the OAS specs. If `git diff` shows
changes, the generated files were stale — commit the regenerated output.

### Build

```sh
make build
```

Compiles the main binary (`CGO_ENABLED=0`, `GOOS=linux`). Fixes any compiler errors
before proceeding.

### Unit tests

```sh
make unittest
```

Runs `go test -v ./... -timeout 20s -failfast` on the main module. Unit tests must
pass entirely on their own — they do not require Docker or network.

If a unit test fails:
- Check for broken assertions, nil dereferences, or race conditions.
- Add the `-race` flag manually if you suspect a race: `go test -race ./...`.

### Integration tests

```sh
# Start the test environment
make compose/up

# Run the tests
make test/integration

# Tear down
make compose/down
```

Integration tests live in `test/suites/integration/`. They run against a live Docker
Compose environment (Kerberos, echo backends, Prometheus, Grafana, Jaeger).

If `make integrationtest` fails:

```sh
# Inspect container logs
make compose/logs

# Tear down before re-running
make compose/down
```

> **Note:** `make integrationtest` uses `-failfast`. Fix the first failing test, then
> re-run the full sequence to surface further failures.

### Security tests

```sh
# Start the TLS-enabled test environment (includes compose-wait internally)
make compose/security/up

# Run the tests
make test/security

# Tear down
make compose/security/down
```

Security tests live in `test/suites/security/`. They verify TLS enforcement, certificate
validation, and other security properties. The environment uses certificates from
`test/certs/`.

If tests fail, inspect logs with:

```sh
make compose/security/logs
```

## Port Reference

| Service | Port |
|---|---|
| Kerberos main API | `30000` |
| Kerberos admin API | `30001` |
| Kerberos metrics | `9464` |
| Echo backend | `15000` |
| Echo metrics | `9463` |
| Prometheus | `9090` |
| Grafana | `3000` |

## Integration Test Structure

**Location:** `test/suites/integration/`

| File | Purpose |
|---|---|
| `main_test.go` | `TestMain` — shared setup (creates orgs, users, groups used across tests) |
| `lib.go` | HTTP helpers (`get`, `post`, `put`, `delete`, `patch`, `trace`, `head`, `options`), response verifiers, test data generators |
| `*_test.go` | Integration tests |
| `client/` | Generated API clients (do not edit by hand) |

**Generated clients** are configured via `client/admin_config.yaml` and
`client/auth_basic_config.yaml`. Regenerate with `make codegen` if the OAS specs change.

**Test flags:**

```
go test -v ./... -count=1 -failfast
```

- `-count=1` disables test result caching
- `-failfast` stops on the first failure
- `-v` prints each test name and result

**Session-based auth:** Tests authenticate by calling a login endpoint and extracting
the `x-krb-session` header. Pass the session to subsequent calls via
`requestEditorSessionID()` from `lib.go`.

## OAS Endpoint Coverage

Every OpenAPI operation **must** have at least one integration test covering its happy
path. Error/edge-case responses (4xx, 5xx) must be covered for state-mutating operations
(POST, PUT, DELETE). Every explicitly documented response code in the OAS spec must be
covered by at least one test.

Before merging a PR that adds or changes an API operation, verify that the endpoint has
coverage in the integration tests.

## FlowComponent Black-Box Coverage

The flow pipeline is the core of Kerberos. Each `FlowComponent` implementor must have
dedicated black-box tests that drive it through its full behaviour via HTTP — not through
internal package calls.

## Adding a New Integration Test

1. Identify which OAS operation(s) and response codes the test covers.
2. Add the test function to the appropriate file in `test/suites/integration/`.
3. Use the HTTP helpers in `lib.go` rather than raw `net/http` calls.
4. Use `username()`, `orgName()`, `groupName()` from `lib.go` for unique identifiers.
5. Call `t.Parallel()` unless the test modifies shared state set up in `TestMain`.
6. Tick the corresponding box(es) in the endpoint coverage and FlowComponent checklists.
7. Run the full workflow (`compose` → `compose-wait` → `integrationtest`) to verify locally.
