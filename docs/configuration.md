# Configuration

Kerberos is configured with a JSON file passed through `--config`.

At startup, configuration goes through:

1. Reference resolution (`${...}` syntax)
2. JSON schema validation
3. Unmarshal into internal config structures

## Reference syntax

Config values support two reference types:

- Environment references: `${env:KEY}` and `${env:KEY:fallback}`
- In-document references: `${ref:path.to.value}`

References are resolved before schema validation, and circular references fail startup.

## Source of truth for structure and fields

Use the JSON schemas as the canonical reference for all supported fields, required keys, and value constraints:

- Root schema: `internal/config/schemas/config_schema.json`
- Section schemas:
  - `internal/config/schemas/gateway_schema.json`
  - `internal/config/schemas/router_schema.json`
  - `internal/config/schemas/auth_schema.json`
  - `internal/config/schemas/oas_schema.json`
  - `internal/config/schemas/observability_schema.json`
  - `internal/config/schemas/persistence_schema.json`
  - `internal/config/schemas/admin_schema.json`
  - `internal/config/schemas/cookies_schema.json`
  - `internal/config/schemas/origins_schema.json`
  - `internal/config/schemas/ordered_schema.json`

## Related docs

- [Routing](./routing.md)
- [Authentication](./authentication.md)
- [Observability](./observability.md)
