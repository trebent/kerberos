---
name: oas
description: 'Design, extend, and regenerate OpenAPI specifications for Kerberos. Covers OAS 3.0.4 conventions used in this repo, the oapi-codegen workflow, and how to implement generated server stubs and test clients.'
argument-hint: 'Optional: the OAS file to work on (admin, auth_basic, gateway) or the operation area (e.g. debug, users, groups)'
---

# OpenAPI Development — Kerberos

## When to Use

- Adding a new endpoint or modifying an existing one in any Kerberos OAS spec
- Regenerating server boilerplate and integration-test client code after an OAS change
- Reviewing an OAS spec for conformance with Kerberos conventions
- Understanding how generated code maps to implementation files

---

## OAS File Locations

| File | API | Serves |
|---|---|---|
| `openapi/admin.yaml` | Administration API | User, group, permission, debug session, flow, and OAS management |
| `openapi/auth_basic.yaml` | Basic Authentication API | Organisation, user, group management for basic-auth backed backends |
| `openapi/gateway.yaml` | Gateway proxy API | HTTP method forwarding to registered backends |

All specs must conform to **OpenAPI 3.0.4** — the last version fully supported by
`oapi-codegen`. Do not upgrade to 3.1.x.

---

## oapi-codegen Configuration

Generated code is produced by `oapi-codegen` v2. Configuration files live next to the
generated output:

### Server/handler stubs (main module)

Located under `internal/oapi/`. Each API has its own `generate.go` file that contains
a `//go:generate` directive referencing the config.

Run codegen for all APIs:

```sh
make codegen
```

This runs `go generate ./...` on the main module and `cd test/suites/integration && go generate ./...` for the test clients.

### Integration-test clients

| Config file | Generated output | Package |
|---|---|---|
| `test/suites/integration/client/admin_config.yaml` | `test/suites/integration/client/admin/client.go` | `adminapi` |
| `test/suites/integration/client/auth_basic_config.yaml` | `test/suites/integration/client/auth/basic/client.go` | `authbasicapi` |

oapi-codegen config schema:

```yaml
# yaml-language-server: $schema=https://raw.githubusercontent.com/oapi-codegen/oapi-codegen/HEAD/configuration-schema.json
package: <go_package_name>
generate:
  client: true   # for test clients
  models: true
  # For server stubs, add: strict-server: true, chi-server: true / std-http-server: true
output: "<relative_path_to_generated_file>"
```

After any OAS change, always run `make codegen` and commit the regenerated files.

---

## OAS 3.0.4 Conventions

### Schema declaration

```yaml
# yaml-language-server: $schema=https://spec.openapis.org/oas/3.0/schema/2024-10-18
openapi: "3.0.4"
info:
  version: <semver>
  title: <title>
  description: |
    <description>
```

### Common types — avoid re-declaration

Declare schemas, request bodies, responses, and headers in `components/` and reference
them with `$ref`. Never inline a type that appears in more than one place.

```yaml
components:
  schemas:
    APIErrorResponse:          # Shared error envelope — reuse across all error responses
      type: object
      additionalProperties: false
      properties:
        errors:
          type: array
          items:
            type: string
          minItems: 1
      required:
        - errors

  requestBodies:
    CreateFooRequest:
      description: Request body for creating a Foo.
      required: true
      content:
        application/json:
          schema:
            type: object
            additionalProperties: false
            properties:
              name:
                type: string
                minLength: 1
            required:
              - name

  responses:
    FooResponse:
      description: A Foo resource.
      content:
        application/json:
          schema:
            $ref: "#/components/schemas/Foo"

  headers:
    sessionid:
      required: true
      description: A session ID.
      schema:
        type: string
```

### State all response codes

Every operation must list **every** response code it can return. Use the shared
`APIErrorResponse` schema for all error responses. Do not use `default` as a catch-all
for documented error cases — `default` is only appropriate for gateway-style catch-all
operations (see `gateway.yaml`).

```yaml
paths:
  /api/admin/foos/{fooID}:
    get:
      operationId: GetFoo
      parameters:
        - name: fooID
          in: path
          required: true
          schema:
            type: integer
      responses:
        "200":
          content:
            application/json:
              schema:
                $ref: "#/components/schemas/Foo"
          description: Got the Foo successfully.
        "400":
          content:
            application/json:
              schema:
                $ref: "#/components/schemas/APIErrorResponse"
          description: Bad request.
        "401":
          content:
            application/json:
              schema:
                $ref: "#/components/schemas/APIErrorResponse"
          description: Unauthorized.
        "403":
          content:
            application/json:
              schema:
                $ref: "#/components/schemas/APIErrorResponse"
          description: Forbidden.
        "404":
          content:
            application/json:
              schema:
                $ref: "#/components/schemas/APIErrorResponse"
          description: Not found.
        "500":
          content:
            application/json:
              schema:
                $ref: "#/components/schemas/APIErrorResponse"
          description: Internal error.
```

### Standard response codes by operation type

| Operation | Expected success code | Common error codes |
|---|---|---|
| Read (GET) | `200` | `400`, `401`, `403`, `404`, `500` |
| Create (POST) | `201` | `400`, `401`, `403`, `409`, `500` |
| Update (PUT) | `204` | `400`, `401`, `403`, `404`, `409`, `500` |
| Delete (DELETE) | `204` | `400`, `401`, `403`, `404`, `500` |
| Login | `204` (with session header) | `400`, `401`, `429`, `500` |
| Logout | `204` | `400`, `401`, `500` |

### Schema constraints

- Always set `additionalProperties: false` on object schemas to prevent undocumented fields.
- Use `minLength: 1` on string fields that must be non-empty.
- Use `minimum`/`maximum` on integer fields where bounds are meaningful.
- Use `format: date-time` for timestamps.
- Use `format: int64` for large integer IDs only when necessary; plain `integer` is fine for small IDs.

### Security

Use global session-based security and exempt only public endpoints:

```yaml
security:
  - sessionid: []

paths:
  /api/admin/superuser/login:
    post:
      security: []   # Public endpoint — no session required
```

### operationId

Every operation must have a unique, descriptive `operationId` in PascalCase:
- `GetFoo`, `CreateFoo`, `UpdateFoo`, `DeleteFoo`, `ListFoos`
- `LoginSuperuser`, `LogoutSuperuser`

---

## End-to-End Workflow for Adding an Endpoint

1. **Edit the OAS spec** (`openapi/admin.yaml`, `openapi/auth_basic.yaml`, or `openapi/gateway.yaml`):
   - Add or update path, parameters, request body, and all response codes.
   - Define new schemas/request bodies/responses in `components/` if they appear in more than one place.
   - Follow all conventions above.

2. **Regenerate code:**

   ```sh
   make codegen
   ```

   This regenerates server stubs, models, and test clients. Review the diff carefully — new operations produce new handler function signatures.

3. **Implement the handler:**
   - Locate the generated strict-server interface in `internal/oapi/<api>/`.
   - Add the new method to the struct that implements the interface (typically in `internal/<api>/`).
   - Wire any new DB queries, business logic, or middleware needed.

4. **Write integration tests:**
   - Add tests to `test/suites/integration/` covering all documented response codes.
   - Consult the endpoint coverage checklist in the `validation` skill.

5. **Validate:**

   ```sh
   make lint
   make unittest
   make compose && make compose-wait && make integrationtest && make compose-down
   ```

---

## FlowComponent OAS Plumbing

When a new OAS spec is added for a backend:

1. Place the spec in `openapi/` (for Kerberos' own APIs) or in `test/oas/` (for backend test stubs).
2. Reference the backend name in the Kerberos config so the OAS validator component picks it up.
3. The OAS validator (`internal/oas`) loads all specs from `OAS_DIRECTORY` at startup.
4. The admin API endpoint `GET /api/admin/oas/{backend}` exposes the loaded spec — verify it is returned correctly.

---

## Useful References

- oapi-codegen docs: https://github.com/oapi-codegen/oapi-codegen
- OAS 3.0.4 spec: https://spec.openapis.org/oas/v3.0.4
- Config schema: https://raw.githubusercontent.com/oapi-codegen/oapi-codegen/HEAD/configuration-schema.json
