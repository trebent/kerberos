---
name: validation
description: 'Validate a Kerberos PR end-to-end: lint, vulncheck, unit tests, integration tests, security tests, OAS endpoint coverage, and FlowComponent black-box coverage. Everything a PR needs to pass CI and merge to main.'
argument-hint: 'Optional: specific validation stage to run (lint, vulncheck, unittest, integrationtest, securitytest, coverage)'
---

# Validation — Kerberos

## When to Use

- Verifying that a code change is ready to merge (all CI checks pass locally)
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
| Postgres DB backend tests | `make test/unit/postgres && make postgres/stop` | All `*_test.go` files tagged with 'postgres_integration' |
| Integration tests | `make compose/up && && make test/integration && make compose/down` | All tests in `test/suites/integration/` against a live integration environment |
| Security tests | `make compose/security/up && make test/security && make compose/security/down` | All tests in `test/suites/security/` against a TLS-enabled environment |
| Connector tests | `make compose/connector/up && make test/connector && make compose/connector/down` | All tests in `test/suites/connector/` against a connector-enabled environment |
| krbctl tests | `make test/krbctl` | All tests in `test/suites/krbctl/` against a TLS-enabled environment |

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
make test/unit/postgres

# 5. Integration tests
make compose/up
make test/integration
make compose/down

# 6. Security tests
make compose/security/up
make test/security
make compose/security/down

# 7. Connector tests
make compose/connector/up
make test/connector
make compose/connector/down

# 8. krbctl tests
make test/krbctl
```
