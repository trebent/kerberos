---
name: integration-testing
description: 'Run, write, and verify integration tests for Kerberos. Use for: running integration tests locally, setting up and tearing down the Docker Compose test environment, debugging test failures with logs, writing new integration tests, and verifying OpenAPI operation coverage. Covers make targets: compose, compose-wait, compose-logs, compose-down, integrationtest.'
argument-hint: 'Optional: feature or API area to focus on (e.g. auth-basic, admin, gateway)'
---

# Integration Testing — Kerberos

## When to Use

- Running integration tests locally against a live Kerberos environment
- Setting up or tearing down the Docker Compose test environment
- Investigating test failures by inspecting container logs
- Writing new integration tests and verifying they cover all OpenAPI operations
- Reviewing whether a change is fully covered by integration tests before opening a PR

---

## Make Targets

| Target | Purpose | When to Run |
|---|---|---|
| `make compose` | Starts the full Docker Compose test environment (Kerberos, echo backends, Prometheus, Grafana) | Before running tests, or after a config change |
| `make compose-wait` | Polls the admin API until Kerberos is healthy (`401` on `/api/admin/flow`) | Always run after `compose` — tests will fail if Kerberos is not ready |
| `make compose-logs` | Streams logs for `kerberos`, `echo`, and `protected-echo` containers | When tests fail or produce unexpected results |
| `make compose-down` | Tears down all containers | After tests complete, or to reset state before re-running |
| `make integrationtest` | Runs all integration tests with `-v ./... -count=1 -failfast` | After `compose-wait` confirms readiness |

---

## Standard Workflow

Run these steps in order every time you execute integration tests locally:

```sh
# 1. Start the test environment
make compose

# 2. Wait until Kerberos is ready (blocks until healthy)
make compose-wait

# 3. Run the integration tests
make integrationtest

# 4. Tear down the environment
make compose-down
```

If `make integrationtest` fails:

```sh
# Inspect logs from relevant containers
make compose-logs

# Then tear down to ensure a clean slate next run
make compose-down
```

> **Note:** `make integrationtest` runs with `-failfast`, so only the first failing test is reported. Fix it and re-run the full sequence to catch further failures.

---

## Port Reference

These are the default local ports, useful when debugging with `curl` or a browser:

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

## Test Structure Conventions

**Location:** `test/integration/`

| File | Purpose |
|---|---|
| `main_test.go` | `TestMain` — shared setup (creates orgs, users, groups used across tests) |
| `lib.go` | HTTP helpers (`get`, `post`, `put`, `delete`, `patch`, `trace`, `head`, `options`), response verifiers, test data generators |
| `admin_api_test.go` | Tests for the Admin API |
| `auth_basic_api_test.go` | Tests for the Basic Auth API |
| `gateway_test.go` | Tests for the Gateway proxy API |
| `client/` | Generated API clients (do not edit by hand) |

**Generated clients** are configured via `client/admin_config.yaml` and `client/auth_basic_config.yaml`. Regenerate with the appropriate `make` target if the OpenAPI specs change.

**Test flags used by `make integrationtest`:**

```
go test -v ./... -count=1 -failfast
```

- `-count=1` disables test result caching — always re-runs against the live environment
- `-failfast` stops on the first failure to surface the root cause quickly
- `-v` prints each test name and result for visibility

**Session-based auth:** Tests authenticate by calling a login endpoint and extracting the `x-krb-session` header. Pass the session to subsequent calls via `requestEditorSessionID()` from `lib.go`.

---

## OpenAPI Coverage Checklist

Every OpenAPI operation defined in Kerberos **must** have at least one integration test covering its happy path. Error/edge-case coverage is expected for operations that handle state mutations (POST, PUT, DELETE).

Before merging a PR that adds or changes an API operation, verify all boxes below are checked:

### Admin API (`openapi/admin.yaml`)

- [ ] `POST /api/admin/superuser/login` — successful login returns session header
- [ ] `POST /api/admin/superuser/login` — failed login (wrong credentials) returns error
- [ ] `POST /api/admin/superuser/logout` — session is invalidated after logout
- [ ] `GET /api/admin/flow` — returns flow components (observability, router, authorizer, oas-validator, forwarder)
- [ ] `GET /api/admin/oas/{backend}` — returns OAS spec for a known backend
- [ ] `GET /api/admin/oas/{backend}` — returns error for unknown or invalid backend

### Basic Auth API (`openapi/auth_basic.yaml`)

#### Organisations

- [ ] `POST /api/auth/basic/organisations` — create organisation (also creates admin user)
- [ ] `POST /api/auth/basic/organisations` — conflict when name already exists
- [ ] `POST /api/auth/basic/organisations` — denied for non-superuser sessions
- [ ] `GET /api/auth/basic/organisations` — list organisations
- [ ] `GET /api/auth/basic/organisations/{orgID}` — get specific organisation
- [ ] `PUT /api/auth/basic/organisations/{orgID}` — update organisation
- [ ] `DELETE /api/auth/basic/organisations/{orgID}` — delete organisation

#### Users

- [ ] `POST /api/auth/basic/organisations/{orgID}/users` — create user
- [ ] `POST /api/auth/basic/organisations/{orgID}/users` — conflict when name already exists in org
- [ ] `GET /api/auth/basic/organisations/{orgID}/users` — list users
- [ ] `GET /api/auth/basic/organisations/{orgID}/users/{userID}` — get specific user
- [ ] `PUT /api/auth/basic/organisations/{orgID}/users/{userID}` — update user (including password change)
- [ ] `PUT /api/auth/basic/organisations/{orgID}/users/{userID}` — old credentials rejected after password change
- [ ] `DELETE /api/auth/basic/organisations/{orgID}/users/{userID}` — delete user
- [ ] `POST /api/auth/basic/organisations/{orgID}/users/{userID}/permissions` — get user permissions

#### Groups

- [ ] `POST /api/auth/basic/organisations/{orgID}/groups` — create group
- [ ] `POST /api/auth/basic/organisations/{orgID}/groups` — conflict when name already exists in org
- [ ] `GET /api/auth/basic/organisations/{orgID}/groups` — list groups
- [ ] `GET /api/auth/basic/organisations/{orgID}/groups/{groupID}` — get specific group
- [ ] `PUT /api/auth/basic/organisations/{orgID}/groups/{groupID}` — update group
- [ ] `DELETE /api/auth/basic/organisations/{orgID}/groups/{groupID}` — delete group

#### User–Group Binding

- [ ] `PUT /api/auth/basic/organisations/{orgID}/users/{userID}/groups` — assign user to groups

#### Cross-cutting

- [ ] Organisation isolation: user in org A cannot access resources in org B

### Gateway API (`openapi/gateway.yaml`)

- [ ] `GET /gw/backend/{backend}/*` — proxied to backend
- [ ] `POST /gw/backend/{backend}/*` — proxied to backend
- [ ] `PUT /gw/backend/{backend}/*` — proxied to backend
- [ ] `DELETE /gw/backend/{backend}/*` — proxied to backend
- [ ] `PATCH /gw/backend/{backend}/*` — proxied to backend
- [ ] `HEAD /gw/backend/{backend}/*` — proxied to backend
- [ ] `OPTIONS /gw/backend/{backend}/*` — proxied to backend
- [ ] `TRACE /gw/backend/{backend}/*` — proxied to backend
- [ ] Any method with unknown `{backend}` — returns `404`
- [ ] Any method with malformed backend identifier — returns `400`

---

## Adding a New Integration Test

1. Identify which OpenAPI operation(s) the test covers.
2. Add the test function to the appropriate file in `test/integration/` (e.g. `auth_basic_api_test.go` for Basic Auth operations).
3. Use the HTTP helpers in `lib.go` rather than raw `net/http` calls.
4. Use `username()`, `orgName()`, `groupName()` from `lib.go` to generate unique identifiers and avoid test pollution.
5. Call `t.Parallel()` unless the test modifies global shared state set up in `TestMain`.
6. Tick the corresponding box(es) in the coverage checklist above.
7. Run the full workflow (`compose` → `compose-wait` → `integrationtest`) to verify locally before pushing.
