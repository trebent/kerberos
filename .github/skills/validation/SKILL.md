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
| Lint | `make lint` | golangci-lint on main module |
| Vulnerability scan | `make vulncheck` | govulncheck on main module |
| Codegen drift | `make codegen` + git diff | Generated code is up-to-date with OAS specs |
| Build | `make go-build` | Main binary compiles |
| Unit tests | `make unittest` | All `*_test.go` files in the main module |
| Integration tests | `make compose && make compose-wait && make integrationtest && make compose-down` | All tests in `test/suites/integration/` against a live environment |
| Security tests | `make compose-security && make securitytest && make compose-security-down` | All tests in `test/suites/security/` against a TLS-enabled environment |

---

## Full Validation Workflow

Run these steps in order to reproduce a complete CI pass locally:

```sh
# 1. Lint and static analysis
make lint
make vulncheck

# 2. Verify generated code is current
make codegen
git diff --exit-code

# 3. Build
make go-build

# 4. Unit tests
make unittest

# 5. Integration tests
make compose
make compose-wait
make integrationtest
make compose-down

# 6. Security tests
make compose-security
make securitytest
make compose-security-down
```

If any step fails, fix it before proceeding. All stages must pass for a PR to merge.

---

## Stage Reference

### Lint

```sh
make lint
```

Runs `golangci-lint run --fix` on the main module. Configuration is in `.golangci.yaml`.
Fix all reported issues before committing.

### Vulnerability scan

```sh
make vulncheck
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
make go-build
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
make compose

# Wait until Kerberos is ready (blocks until 401 on /api/admin/flow)
make compose-wait

# Run the tests
make integrationtest

# Tear down
make compose-down
```

Integration tests live in `test/suites/integration/`. They run against a live Docker
Compose environment (Kerberos, echo backends, Prometheus, Grafana, Jaeger).

If `make integrationtest` fails:

```sh
# Inspect container logs
make compose-logs

# Tear down before re-running
make compose-down
```

> **Note:** `make integrationtest` uses `-failfast`. Fix the first failing test, then
> re-run the full sequence to surface further failures.

### Security tests

```sh
# Start the TLS-enabled test environment (includes compose-wait internally)
make compose-security

# Run the tests
make securitytest

# Tear down
make compose-security-down
```

Security tests live in `test/suites/security/`. They verify TLS enforcement, certificate
validation, and other security properties. The environment uses certificates from
`test/certs/`.

If tests fail, inspect logs with:

```sh
make compose-logs-security
```

---

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

---

## Integration Test Structure

**Location:** `test/suites/integration/`

| File | Purpose |
|---|---|
| `main_test.go` | `TestMain` — shared setup (creates orgs, users, groups used across tests) |
| `lib.go` | HTTP helpers (`get`, `post`, `put`, `delete`, `patch`, `trace`, `head`, `options`), response verifiers, test data generators |
| `admin_api_test.go` | Tests for Admin API superuser/session endpoints |
| `admin_api_users_test.go` | Tests for Admin API user management endpoints |
| `admin_api_groups_test.go` | Tests for Admin API group management endpoints |
| `admin_api_group_bindings_test.go` | Tests for Admin API user–group binding endpoints |
| `admin_api_permissions_test.go` | Tests for Admin API permission endpoints |
| `auth_basic_api_test.go` | Tests for Basic Auth API session endpoints |
| `auth_basic_api_organisations_test.go` | Tests for Basic Auth organisations endpoints |
| `auth_basic_api_users_test.go` | Tests for Basic Auth users endpoints |
| `auth_basic_api_groups_test.go` | Tests for Basic Auth groups endpoints |
| `auth_basic_api_bindings_test.go` | Tests for Basic Auth user–group binding endpoints |
| `auth_basic_test.go` | Cross-cutting auth and isolation tests |
| `gateway_test.go` | Tests for Gateway proxy API |
| `metrics_test.go` | Tests for Prometheus metrics exposure |
| `tracing_test.go` | Tests for distributed tracing |
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

---

## OAS Endpoint Coverage

Every OpenAPI operation **must** have at least one integration test covering its happy
path. Error/edge-case responses (4xx, 5xx) must be covered for state-mutating operations
(POST, PUT, DELETE). Every explicitly documented response code in the OAS spec must be
covered by at least one test.

Before merging a PR that adds or changes an API operation, verify all boxes below are
checked for the affected API.

### Admin API (`openapi/admin.yaml`)

#### Superuser

- [ ] `POST /api/admin/superuser/login` — `204` successful login, session header returned
- [ ] `POST /api/admin/superuser/login` — `400` bad request (malformed body)
- [ ] `POST /api/admin/superuser/login` — `401` wrong credentials
- [ ] `POST /api/admin/superuser/login` — `429` rate limited
- [ ] `POST /api/admin/superuser/logout` — `204` session invalidated
- [ ] `POST /api/admin/superuser/logout` — `403` no active session

#### Admin user login/logout

- [ ] `POST /api/admin/login` — `204` successful login, session header returned
- [ ] `POST /api/admin/login` — `400` bad request
- [ ] `POST /api/admin/login` — `401` wrong credentials
- [ ] `POST /api/admin/logout` — `204` session invalidated
- [ ] `POST /api/admin/logout` — `400` no session
- [ ] `POST /api/admin/logout` — `401` unauthorized

#### Admin users

- [ ] `POST /api/admin/users` — `201` user created
- [ ] `POST /api/admin/users` — `400` bad request
- [ ] `POST /api/admin/users` — `401` unauthorized
- [ ] `POST /api/admin/users` — `403` forbidden
- [ ] `POST /api/admin/users` — `409` conflict (duplicate username)
- [ ] `GET /api/admin/users` — `200` list of users
- [ ] `GET /api/admin/users` — `401` unauthorized
- [ ] `GET /api/admin/users` — `403` forbidden
- [ ] `GET /api/admin/users/{userID}` — `200` specific user
- [ ] `GET /api/admin/users/{userID}` — `400` bad request
- [ ] `GET /api/admin/users/{userID}` — `401` unauthorized
- [ ] `GET /api/admin/users/{userID}` — `403` forbidden
- [ ] `GET /api/admin/users/{userID}` — `404` not found
- [ ] `PUT /api/admin/users/{userID}` — `204` updated
- [ ] `PUT /api/admin/users/{userID}` — `400` bad request
- [ ] `PUT /api/admin/users/{userID}` — `401` unauthorized
- [ ] `PUT /api/admin/users/{userID}` — `403` forbidden
- [ ] `PUT /api/admin/users/{userID}` — `404` not found
- [ ] `PUT /api/admin/users/{userID}` — `409` conflict
- [ ] `DELETE /api/admin/users/{userID}` — `204` deleted
- [ ] `DELETE /api/admin/users/{userID}` — `400` bad request
- [ ] `DELETE /api/admin/users/{userID}` — `401` unauthorized
- [ ] `DELETE /api/admin/users/{userID}` — `403` forbidden
- [ ] `DELETE /api/admin/users/{userID}` — `404` not found
- [ ] `PUT /api/admin/users/{userID}/password` — `204` password changed
- [ ] `PUT /api/admin/users/{userID}/password` — `400` bad request
- [ ] `PUT /api/admin/users/{userID}/password` — `401` unauthorized
- [ ] `PUT /api/admin/users/{userID}/password` — `403` forbidden
- [ ] `PUT /api/admin/users/{userID}/password` — `404` not found
- [ ] `PUT /api/admin/users/{userID}/groups` — `204` groups updated
- [ ] `PUT /api/admin/users/{userID}/groups` — `400` bad request
- [ ] `PUT /api/admin/users/{userID}/groups` — `401` unauthorized
- [ ] `PUT /api/admin/users/{userID}/groups` — `403` forbidden
- [ ] `PUT /api/admin/users/{userID}/groups` — `404` not found

#### Admin groups

- [ ] `POST /api/admin/groups` — `201` group created
- [ ] `POST /api/admin/groups` — `400` bad request
- [ ] `POST /api/admin/groups` — `401` unauthorized
- [ ] `POST /api/admin/groups` — `403` forbidden
- [ ] `POST /api/admin/groups` — `409` conflict (duplicate name)
- [ ] `GET /api/admin/groups` — `200` list of groups
- [ ] `GET /api/admin/groups` — `401` unauthorized
- [ ] `GET /api/admin/groups` — `403` forbidden
- [ ] `GET /api/admin/groups/{groupID}` — `200` specific group
- [ ] `GET /api/admin/groups/{groupID}` — `401` unauthorized
- [ ] `GET /api/admin/groups/{groupID}` — `403` forbidden
- [ ] `GET /api/admin/groups/{groupID}` — `404` not found
- [ ] `PUT /api/admin/groups/{groupID}` — `204` updated
- [ ] `PUT /api/admin/groups/{groupID}` — `400` bad request
- [ ] `PUT /api/admin/groups/{groupID}` — `401` unauthorized
- [ ] `PUT /api/admin/groups/{groupID}` — `403` forbidden
- [ ] `PUT /api/admin/groups/{groupID}` — `404` not found
- [ ] `PUT /api/admin/groups/{groupID}` — `409` conflict
- [ ] `DELETE /api/admin/groups/{groupID}` — `204` deleted
- [ ] `DELETE /api/admin/groups/{groupID}` — `400` bad request
- [ ] `DELETE /api/admin/groups/{groupID}` — `401` unauthorized
- [ ] `DELETE /api/admin/groups/{groupID}` — `403` forbidden
- [ ] `DELETE /api/admin/groups/{groupID}` — `404` not found

#### Admin permissions

- [ ] `GET /api/admin/permissions` — `200` list of permissions
- [ ] `GET /api/admin/permissions` — `401` unauthorized

#### Admin debug sessions

- [ ] `POST /api/admin/debug/{backend}/sessions` — `200` session started
- [ ] `POST /api/admin/debug/{backend}/sessions` — `400` bad request
- [ ] `POST /api/admin/debug/{backend}/sessions` — `404` backend not found
- [ ] `POST /api/admin/debug/{backend}/sessions` — `409` already in debug mode
- [ ] `GET /api/admin/debug/{backend}/sessions` — `200` list of sessions
- [ ] `GET /api/admin/debug/{backend}/sessions` — `404` backend not found
- [ ] `GET /api/admin/debug/{backend}/sessions/{sessionId}` — `200` specific session
- [ ] `GET /api/admin/debug/{backend}/sessions/{sessionId}` — `400` bad request
- [ ] `GET /api/admin/debug/{backend}/sessions/{sessionId}` — `404` not found
- [ ] `PUT /api/admin/debug/{backend}/sessions/{sessionId}` — `200` extended
- [ ] `PUT /api/admin/debug/{backend}/sessions/{sessionId}` — `400` bad request
- [ ] `PUT /api/admin/debug/{backend}/sessions/{sessionId}` — `404` not found
- [ ] `PUT /api/admin/debug/{backend}/sessions/{sessionId}` — `409` not in debug mode
- [ ] `DELETE /api/admin/debug/{backend}/sessions/{sessionId}` — `204` stopped
- [ ] `DELETE /api/admin/debug/{backend}/sessions/{sessionId}` — `400` bad request
- [ ] `DELETE /api/admin/debug/{backend}/sessions/{sessionId}` — `404` not found
- [ ] `GET /api/admin/debug/{backend}/sessions/{sessionId}/operations` — `200` list of operations
- [ ] `GET /api/admin/debug/{backend}/sessions/{sessionId}/operations` — `400` bad request
- [ ] `GET /api/admin/debug/{backend}/sessions/{sessionId}/operations` — `404` not found

#### Flow and OAS

- [ ] `GET /api/admin/flow` — `200` flow components returned
- [ ] `GET /api/admin/flow` — `401` unauthorized
- [ ] `GET /api/admin/flow` — `403` forbidden
- [ ] `GET /api/admin/oas/{backend}` — `200` OAS spec returned for known backend
- [ ] `GET /api/admin/oas/{backend}` — `401` unauthorized
- [ ] `GET /api/admin/oas/{backend}` — `403` forbidden
- [ ] `GET /api/admin/oas/{backend}` — `404` backend not found

### Basic Auth API (`openapi/auth_basic.yaml`)

#### Organisations

- [ ] `POST /api/auth/basic/organisations` — `201` organisation created (also creates admin user)
- [ ] `POST /api/auth/basic/organisations` — `400` bad request
- [ ] `POST /api/auth/basic/organisations` — `401` unauthorized
- [ ] `POST /api/auth/basic/organisations` — `409` conflict (duplicate name)
- [ ] `GET /api/auth/basic/organisations` — `200` list organisations
- [ ] `GET /api/auth/basic/organisations` — `401` unauthorized
- [ ] `GET /api/auth/basic/organisations/{orgID}` — `200` specific organisation
- [ ] `GET /api/auth/basic/organisations/{orgID}` — `400` bad request
- [ ] `GET /api/auth/basic/organisations/{orgID}` — `401` unauthorized
- [ ] `GET /api/auth/basic/organisations/{orgID}` — `404` not found
- [ ] `PUT /api/auth/basic/organisations/{orgID}` — `204` updated
- [ ] `PUT /api/auth/basic/organisations/{orgID}` — `400` bad request
- [ ] `PUT /api/auth/basic/organisations/{orgID}` — `401` unauthorized
- [ ] `PUT /api/auth/basic/organisations/{orgID}` — `404` not found
- [ ] `DELETE /api/auth/basic/organisations/{orgID}` — `204` deleted
- [ ] `DELETE /api/auth/basic/organisations/{orgID}` — `400` bad request
- [ ] `DELETE /api/auth/basic/organisations/{orgID}` — `401` unauthorized
- [ ] `DELETE /api/auth/basic/organisations/{orgID}` — `404` not found

#### Users

- [ ] `POST /api/auth/basic/organisations/{orgID}/users` — `201` user created
- [ ] `POST /api/auth/basic/organisations/{orgID}/users` — `400` bad request
- [ ] `POST /api/auth/basic/organisations/{orgID}/users` — `401` unauthorized
- [ ] `POST /api/auth/basic/organisations/{orgID}/users` — `404` org not found
- [ ] `POST /api/auth/basic/organisations/{orgID}/users` — `409` conflict (duplicate name in org)
- [ ] `GET /api/auth/basic/organisations/{orgID}/users` — `200` list users
- [ ] `GET /api/auth/basic/organisations/{orgID}/users` — `401` unauthorized
- [ ] `GET /api/auth/basic/organisations/{orgID}/users` — `404` org not found
- [ ] `GET /api/auth/basic/organisations/{orgID}/users/{userID}` — `200` specific user
- [ ] `GET /api/auth/basic/organisations/{orgID}/users/{userID}` — `400` bad request
- [ ] `GET /api/auth/basic/organisations/{orgID}/users/{userID}` — `401` unauthorized
- [ ] `GET /api/auth/basic/organisations/{orgID}/users/{userID}` — `404` not found
- [ ] `PUT /api/auth/basic/organisations/{orgID}/users/{userID}` — `204` updated
- [ ] `PUT /api/auth/basic/organisations/{orgID}/users/{userID}` — `400` bad request
- [ ] `PUT /api/auth/basic/organisations/{orgID}/users/{userID}` — `401` unauthorized
- [ ] `PUT /api/auth/basic/organisations/{orgID}/users/{userID}` — `404` not found
- [ ] `DELETE /api/auth/basic/organisations/{orgID}/users/{userID}` — `204` deleted
- [ ] `DELETE /api/auth/basic/organisations/{orgID}/users/{userID}` — `400` bad request
- [ ] `DELETE /api/auth/basic/organisations/{orgID}/users/{userID}` — `401` unauthorized
- [ ] `DELETE /api/auth/basic/organisations/{orgID}/users/{userID}` — `404` not found
- [ ] `POST /api/auth/basic/organisations/{orgID}/users/{userID}/permissions` — user permissions returned

#### Groups

- [ ] `POST /api/auth/basic/organisations/{orgID}/groups` — `201` group created
- [ ] `POST /api/auth/basic/organisations/{orgID}/groups` — `400` bad request
- [ ] `POST /api/auth/basic/organisations/{orgID}/groups` — `401` unauthorized
- [ ] `POST /api/auth/basic/organisations/{orgID}/groups` — `404` org not found
- [ ] `POST /api/auth/basic/organisations/{orgID}/groups` — `409` conflict (duplicate name in org)
- [ ] `GET /api/auth/basic/organisations/{orgID}/groups` — `200` list groups
- [ ] `GET /api/auth/basic/organisations/{orgID}/groups` — `401` unauthorized
- [ ] `GET /api/auth/basic/organisations/{orgID}/groups` — `404` org not found
- [ ] `GET /api/auth/basic/organisations/{orgID}/groups/{groupID}` — `200` specific group
- [ ] `GET /api/auth/basic/organisations/{orgID}/groups/{groupID}` — `400` bad request
- [ ] `GET /api/auth/basic/organisations/{orgID}/groups/{groupID}` — `401` unauthorized
- [ ] `GET /api/auth/basic/organisations/{orgID}/groups/{groupID}` — `404` not found
- [ ] `PUT /api/auth/basic/organisations/{orgID}/groups/{groupID}` — `204` updated
- [ ] `PUT /api/auth/basic/organisations/{orgID}/groups/{groupID}` — `400` bad request
- [ ] `PUT /api/auth/basic/organisations/{orgID}/groups/{groupID}` — `401` unauthorized
- [ ] `PUT /api/auth/basic/organisations/{orgID}/groups/{groupID}` — `404` not found
- [ ] `PUT /api/auth/basic/organisations/{orgID}/groups/{groupID}` — `409` conflict
- [ ] `DELETE /api/auth/basic/organisations/{orgID}/groups/{groupID}` — `204` deleted
- [ ] `DELETE /api/auth/basic/organisations/{orgID}/groups/{groupID}` — `400` bad request
- [ ] `DELETE /api/auth/basic/organisations/{orgID}/groups/{groupID}` — `401` unauthorized
- [ ] `DELETE /api/auth/basic/organisations/{orgID}/groups/{groupID}` — `404` not found

#### User–Group Binding

- [ ] `PUT /api/auth/basic/organisations/{orgID}/users/{userID}/groups` — `204` groups assigned
- [ ] `PUT /api/auth/basic/organisations/{orgID}/users/{userID}/groups` — `400` bad request
- [ ] `PUT /api/auth/basic/organisations/{orgID}/users/{userID}/groups` — `401` unauthorized
- [ ] `PUT /api/auth/basic/organisations/{orgID}/users/{userID}/groups` — `404` not found

#### Cross-cutting

- [ ] Organisation isolation: a user in org A cannot access resources protected by org B

### Gateway API (`openapi/gateway.yaml`)

- [ ] `GET /gw/backend/{backend}/*` — `200` proxied to backend (echo)
- [ ] `POST /gw/backend/{backend}/*` — `200` proxied to backend (echo)
- [ ] `PUT /gw/backend/{backend}/*` — `200` proxied to backend (echo)
- [ ] `DELETE /gw/backend/{backend}/*` — `200` proxied to backend (echo)
- [ ] `PATCH /gw/backend/{backend}/*` — `200` proxied to backend (echo)
- [ ] `HEAD /gw/backend/{backend}/*` — `200` proxied to backend (echo)
- [ ] `OPTIONS /gw/backend/{backend}/*` — `200` proxied to backend (echo)
- [ ] `TRACE /gw/backend/{backend}/*` — `200` proxied to backend (echo)
- [ ] Any method with unknown `{backend}` — `404` backend not found
- [ ] Any method with malformed `{backend}` identifier — `400` bad request

---

## FlowComponent Black-Box Coverage

The flow pipeline is the core of Kerberos. Each `FlowComponent` implementor must have
dedicated black-box tests that drive it through its full behaviour via HTTP — not through
internal package calls.

### Component inventory

| Package | Component | What to test |
|---|---|---|
| `internal/composer/observability` | `obs` | Metrics incremented for each request; trace span created and propagated |
| `internal/composer/router` | `router` | Valid backend routed correctly; unknown backend → `404`; malformed backend identifier → `400` |
| `internal/auth` | `auth` (authorizer) | Authenticated request forwarded; unauthenticated request rejected with `401`; group-restricted path rejected with `403` |
| `internal/oas` | `validator` | Request matching OAS spec passes; request violating OAS spec rejected with `400` |
| `internal/composer/forwarder` | `forwarder` | Request forwarded to correct backend URL; backend unreachable → `502` or `504` |

### Coverage checklist

- [ ] `observability`: request metrics (counter, latency histogram) are emitted for a proxied request
- [ ] `observability`: trace context is propagated to the upstream backend
- [ ] `router`: happy-path routing for each configured backend
- [ ] `router`: `404` returned for an unknown backend name
- [ ] `router`: `400` returned for a path that does not match the routing pattern
- [ ] `auth`: unauthenticated request to a protected backend returns `401`
- [ ] `auth`: authenticated request with insufficient group membership returns `403`
- [ ] `auth`: authenticated request with correct group membership is forwarded
- [ ] `auth`: exempt paths bypass authentication as configured
- [ ] `oas`: request that satisfies the backend OAS spec is forwarded
- [ ] `oas`: request with missing required fields returns `400`
- [ ] `oas`: request with an unknown path returns `404`
- [ ] `forwarder`: response body and status code from the backend are faithfully proxied
- [ ] `forwarder`: all HTTP methods are proxied (GET, POST, PUT, DELETE, PATCH, HEAD, OPTIONS, TRACE)

---

## Adding a New Integration Test

1. Identify which OAS operation(s) and response codes the test covers.
2. Add the test function to the appropriate file in `test/suites/integration/`.
3. Use the HTTP helpers in `lib.go` rather than raw `net/http` calls.
4. Use `username()`, `orgName()`, `groupName()` from `lib.go` for unique identifiers.
5. Call `t.Parallel()` unless the test modifies shared state set up in `TestMain`.
6. Tick the corresponding box(es) in the endpoint coverage and FlowComponent checklists.
7. Run the full workflow (`compose` → `compose-wait` → `integrationtest`) to verify locally.
