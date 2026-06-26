---
name: configuration
description: 'Work with Kerberos configuration: add or modify config sections, write JSON schemas, use environment variable and path references, and wire new config into the parser and startup code.'
argument-hint: 'Optional: the config section to work on (gateway, observability, admin, auth, oas, persistence) or the task (add-section, modify-schema, add-reference)'
---

# Configuration — Kerberos

## When to Use

- Adding a new config section for a new feature
- Modifying an existing config schema or adding fields
- Using `${env:...}` or `${ref:...}` references in config
- Understanding how config is parsed and validated at startup
- Debugging config validation errors

See [`docs/configuration.md`](../../docs/configuration.md) for a full reference including annotated examples.

---

## Config Loading Pipeline

At startup, `config.RootConfig` processes the JSON file in four steps:

1. **Escape references** — raw `${...}` tokens that aren't already quoted strings are wrapped in quotes so they survive JSON parsing.
2. **Resolve references** — `${env:...}` tokens are replaced with environment variable values; `${ref:...}` tokens are replaced by the value at the given JSON path in the same file.
3. **Validate schema** — the resolved JSON is validated against the embedded JSON Schema (Draft 7). All `additionalProperties: false`, so unknown fields are errors.
4. **Unmarshal** — the resolved JSON is unmarshalled into the Go struct.

---

## Reference Syntax

| Syntax | Meaning |
|---|---|
| `"${env:VAR}"` | Value of environment variable `VAR`. Error at startup if not set. |
| `"${env:VAR:default}"` | Value of `VAR`, or `"default"` if not set. |
| `"${ref:some.path[0].field}"` | Value at the given JSON path within the same config file. |

References can be nested: a path reference can point to a field that is itself an env reference. Circular references (`${ref:a}` → `${ref:b}` → `${ref:a}`) are detected and cause a startup error.

**Important:** The entire field value must be the reference token — you cannot embed a reference inside a longer string like `"prefix-${env:VAR}-suffix"`.

---

## Adding a New Config Section

Follow these steps whenever a new feature needs configuration.

### 1. Define the Go type

Add the struct to `internal/config/types.go`:

```go
MyFeatureConfig struct {
    Order     int    `json:"order"`
    SomeField string `json:"someField"`
}
```

If the component is a custom-block flow component, include `Order int` and reference the `ordered_schema.json` in the schema (see step 2).

### 2. Add a JSON Schema file

Create `internal/config/schemas/myfeature_schema.json`:

```json
{
  "$schema": "http://json-schema.org/draft-07/schema#",
  "$id": "http://trebent.com/kerberos/schemas/myfeature_schema.json",
  "type": "object",
  "properties": {
    "order": {
      "$ref": "http://trebent.com/kerberos/schemas/ordered_schema.json"
    },
    "someField": {
      "type": "string",
      "minLength": 1,
      "description": "A required non-empty string."
    }
  },
  "required": ["someField"],
  "additionalProperties": false
}
```

Rules:
- Always set `"additionalProperties": false`.
- Use `"description"` on every field.
- Use `"$ref": "http://trebent.com/kerberos/schemas/ordered_schema.json"` for any `order` field used by custom-block components.
- Use `"minLength": 1` on required string fields.
- Use `"minimum"`/`"maximum"` on integer fields where bounds are meaningful.

### 3. Embed the schema

In `internal/config/config.go`, add an embed directive alongside the others:

```go
//go:embed schemas/myfeature_schema.json
schemaBytesMyFeature []byte
```

Then register it in `validateSchema()` by adding it to the `sl.AddSchemas(...)` call:

```go
gojsonschema.NewBytesLoader(schemaBytesMyFeature),
```

### 4. Reference the sub-schema from the root schema

In `internal/config/schemas/config_schema.json`, add a property under `properties`:

```json
"myfeature": {
  "$ref": "http://trebent.com/kerberos/schemas/myfeature_schema.json"
}
```

If the section is optional (most feature sections are), do **not** add it to `required`.

### 5. Add the field to `RootConfig`

In `internal/config/config.go`:

```go
type RootConfig struct {
    // ... existing fields ...
    *MyFeatureConfig `json:"myfeature,omitempty"`
}
```

Add a `MyFeatureEnabled() bool` helper if the feature is optional:

```go
func (rc *RootConfig) MyFeatureEnabled() bool {
    return rc.MyFeatureConfig != nil
}
```

### 6. Add defaults if needed

If the config section has default values (like observability does), initialise them in `config.New()`:

```go
func New() *RootConfig {
    return &RootConfig{
        // ...
        MyFeatureConfig: newMyFeatureConfig(),
    }
}

func newMyFeatureConfig() *MyFeatureConfig {
    return &MyFeatureConfig{SomeField: "default-value"}
}
```

Also add a `postProcess()` method (even if empty) and call it from `(rc *RootConfig).postProcess()`.

### 7. Wire into `main.go`

```go
if cfg.MyFeatureEnabled() {
    comp := myfeature.NewComponent(&myfeature.Opts{Cfg: cfg.MyFeatureConfig})
    customFlowComponents = append(customFlowComponents, comp)
}
```

---

## Schema Conventions

| Convention | Reason |
|---|---|
| `"additionalProperties": false` on all objects | Prevents silent config drift; typos cause startup errors |
| `"$id": "http://trebent.com/kerberos/schemas/<name>_schema.json"` | Required for cross-schema `$ref` resolution |
| `"$ref": "http://trebent.com/kerberos/schemas/ordered_schema.json"` for `order` | Reuse consistent definition for custom-block ordering |
| All string IDs/names use `"minLength": 1` | Prevents empty-string values being accepted |
| Draft 7 (`$schema: http://json-schema.org/draft-07/schema#`) | The schema loader is configured for Draft 7 |

### `ordered_schema.json`

Use this ref for any `order` field in a custom-block component's config schema. It defines `type: integer, minimum: 1` and documents that lower values run first. It ensures consistent behaviour and documentation.

---

## Existing Config Sections Reference

| Section | Required | Config struct | Go enabled check |
|---|---|---|---|
| `gateway` | **yes** | `GatewayConfig` | always enabled |
| `observability` | no | `ObservabilityConfig` | always present (defaults to enabled) |
| `admin` | no | `AdminConfig` | always present (defaults applied) |
| `auth` | no | `AuthConfig` | `cfg.AuthEnabled()` |
| `oas` | no | `OASConfig` | `cfg.OASEnabled()` |
| `persistence` | no | `PersistenceConfig` | always present (defaults to SQLite) |

---

## Useful File Locations

| File | Purpose |
|---|---|
| `internal/config/config.go` | `RootConfig`, schema loading, reference resolution, parse pipeline |
| `internal/config/types.go` | All config structs and defaults |
| `internal/config/schemas/config_schema.json` | Root schema wiring all sub-schemas |
| `internal/config/schemas/ordered_schema.json` | Reusable `order` field schema |
| `internal/config/schemas/` | All sub-schemas (one per config section) |
| `internal/config/config_test.go` | Reference resolution and validation tests |
