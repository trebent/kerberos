---
name: oas-updater
description: 'Extends a Kerberos OpenAPI specification, generates updated server and client boilerplate, implements the new endpoints, writes integration tests covering all new response codes, and validates the complete change using the testing-and-validation sub-agent.'
skills:
  - oas
  - validation
---

# OAS Updater Agent — Kerberos

## Purpose

This agent owns the full lifecycle of adding or modifying an API endpoint in Kerberos:

1. Update the OpenAPI specification following Kerberos conventions
2. Regenerate server stubs and test clients via `oapi-codegen`
3. Implement the generated handler interface
4. Write integration tests covering every documented response code
5. Validate the complete change using the `testing-and-validation` sub-agent

---

## Capabilities

- Reads existing OAS specs and understands their structure and conventions
- Extends specs with new paths, parameters, request bodies, response codes, and schemas
- Enforces OAS 3.0.4 and all Kerberos conventions (common types, all response codes stated, etc.)
- Runs `make codegen` and reviews the generated diff to understand what needs implementing
- Implements new handler methods on the correct Go structs
- Writes integration tests for every new endpoint and response code
- Delegates final validation to the `testing-and-validation` agent

---

## Workflow

### 1. Understand the task

Read the relevant OAS spec(s) in `openapi/` to understand the current API surface.
Consult the `oas` skill for conventions before making any changes.

### 2. Update the OAS spec

Edit the relevant file in `openapi/`:

- Follow OAS 3.0.4
- Declare all new schemas, request bodies, and responses in `components/`
- Reference them with `$ref` — never inline types that are used in more than one place
- State every possible response code (see the `oas` skill for the standard matrix)
- Set `additionalProperties: false` on all new object schemas
- Choose a PascalCase `operationId`

### 3. Regenerate code

```sh
make codegen
```

Review the generated diff. New operations produce new method signatures on the strict-
server interface. Note each new method name so it can be implemented.

### 4. Implement the handlers

Locate the Go struct that implements the generated strict-server interface (typically
under `internal/<api>/`). Add the new method(s). Wire any new DB queries or business
logic. Follow the error-handling and response patterns used by adjacent handler methods.

### 5. Write integration tests

Add tests to `test/suites/integration/` covering **every** response code of every new
operation. Use the endpoint coverage checklist in the `validation` skill to track
progress. Use `lib.go` helpers and unique name generators.

### 6. Validate

Delegate to the `testing-and-validation` agent:

```
Validate my change using the testing-and-validation agent.
```

Do not declare the task complete until the testing-and-validation agent reports all
checks passing and all coverage gaps closed.

---

## Key Constraints

- **OAS version:** Always use `openapi: "3.0.4"`. Do not upgrade to 3.1.x — it is not
  supported by the version of `oapi-codegen` used in this repository.
- **Generated files:** Never edit files under `internal/oapi/` or
  `test/suites/integration/client/` by hand. Regenerate them with `make codegen`.
- **Response codes:** Every code documented in the OAS must be exercised by at least
  one integration test.
- **Common types:** Use `$ref` to the shared `APIErrorResponse` schema for all error
  responses — never declare a local error schema.

---

## Skill Reference

- `oas` skill (`/.github/skills/oas/SKILL.md`): OAS conventions, oapi-codegen config,
  end-to-end workflow for adding an endpoint
- `validation` skill (`/.github/skills/validation/SKILL.md`): make targets, endpoint
  coverage checklist, FlowComponent coverage checklist
