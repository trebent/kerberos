---
name: configuration
description: 'Work with Kerberos configuration: add or modify config sections, write JSON schemas, use environment variable and path references, wire new config into the parser and startup code, and document it in the one page that owns it.'
argument-hint: 'Optional: the config section to work on (gateway, observability, admin, auth, oas, persistence) or the task (add-section, modify-schema, add-reference)'
---

# Configuration — Kerberos

## When to Use

- Adding a new config section for a new feature
- Modifying an existing config schema or adding fields
- Changing a default value
- Using `${env:...}` or `${ref:...}` references in config
- Debugging config validation errors

## Where things are documented

Do not copy config facts into this skill or into other docs. Each one has a single home:

| Topic | Source of truth |
|---|---|
| Loading pipeline, `${env:...}` / `${ref:...}` syntax, validation errors | [`docs/configuration.md`](../../../docs/configuration.md) |
| Top-level keys and the page that owns each section | [`docs/configuration.md#root-keys`](../../../docs/configuration.md#root-keys) |
| Keys, types and defaults of a section | The page listed for it in the root keys table |
| Environment variables | [`docs/deployment.md#environment-variables`](../../../docs/deployment.md#environment-variables) |
| How the docs are kept in step with the code | [`docs/dev/README.md#keeping-the-docs-current`](../../../docs/dev/README.md#keeping-the-docs-current) |

---

## Adding a New Config Section

Follow these steps whenever a new feature needs configuration.

### 1. Define the Go type

Add the struct to `internal/config/kerberos.go` (or `internal/config/shared.go` if the
admin-connector config uses it too):

```go
MyFeature struct {
    Order     int    `json:"order"`
    SomeField string `json:"someField"`
}
```

If the component is a custom-block flow component, include `Order int` and reference `ordered_schema.json` in the schema (see step 2).

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

### 3. Embed and register the schema

In `internal/config/kerberos.go`, add an embed directive alongside the others:

```go
//go:embed schemas/myfeature_schema.json
var myFeatureSchema []byte
```

Then add it to `getKerberosSupportingSchemas()`:

```go
gojsonschema.NewBytesLoader(myFeatureSchema),
```

### 4. Reference the sub-schema from the root schema

In `internal/config/schemas/config_schema.json`, add a property under `properties`:

```json
"myfeature": {
  "$ref": "http://trebent.com/kerberos/schemas/myfeature_schema.json"
}
```

If the section is optional (most feature sections are), do **not** add it to `required`.

### 5. Add the field to `Kerberos`

In `internal/config/kerberos.go`:

```go
type Kerberos struct {
    // ... existing fields ...
    MyFeatureCfg *MyFeature `json:"myfeature,omitempty"`
}
```

Add a `MyFeatureEnabled() bool` helper if the feature is optional:

```go
func (kc *Kerberos) MyFeatureEnabled() bool {
    return kc.MyFeatureCfg != nil
}
```

### 6. Add defaults if needed

Defaults are applied in code, not from the schema's `default` fields. Initialise section defaults in
`NewKerberos()`, and defaults that depend on the parsed file (such as per-item values) in a
`postProcess()` method called from `(*Kerberos).PostProcess()`.

### 7. Wire into `main.go`

```go
if cfg.MyFeatureEnabled() {
    comp := myfeature.NewComponent(&myfeature.Opts{Cfg: cfg.MyFeatureCfg})
    customFlowComponents = append(customFlowComponents, comp)
}
```

### 8. Document it in exactly one place

1. Add a row for the new top-level key to the root keys table in `docs/configuration.md`, linking to the page that documents the feature.
2. Document every key, its type and its default on that page only. Other pages link there instead of repeating values.
3. If the section needs a new page, add it to the index in `docs/README.md`.

The same applies when you add or rename a key in an existing section, or change a default: update the owning page.

---

## Schema Conventions

| Convention | Reason |
|---|---|
| `"additionalProperties": false` on all objects, as a sibling of `properties` | Prevents silent config drift; typos cause startup errors |
| `"$id": "http://trebent.com/kerberos/schemas/<name>_schema.json"` | Required for cross-schema `$ref` resolution |
| `"$ref": "http://trebent.com/kerberos/schemas/ordered_schema.json"` for `order` | Reuse consistent definition for custom-block ordering |
| `"description"` on every field | Documents the key in the schema itself |
| All string IDs/names use `"minLength": 1` | Prevents empty-string values being accepted |
| `"minimum"`/`"maximum"` on integers where bounds are meaningful | Catches out-of-range values at startup |
| Draft 7 (`$schema: http://json-schema.org/draft-07/schema#`) | The schema loader is configured for Draft 7 |

---

## Useful File Locations

| File | Purpose |
|---|---|
| `internal/config/kerberos.go` | `Kerberos` root struct, section structs, schema embeds, `NewKerberos()` defaults, `PostProcess()` |
| `internal/config/shared.go` | Structs and schemas shared with the admin-connector (`Cookies`, `Origins`, `ServerTLS`, `Persistence`) |
| `internal/config/connector.go` | The admin-connector config |
| `internal/config/schemas/` | All schemas (one per config section, plus shared ones) |
| `internal/config/kerberos_test.go` | Parsing, reference and validation tests |
