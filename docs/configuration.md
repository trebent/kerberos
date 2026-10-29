# Configuration

Kerberos reads its settings from one JSON file that you name at startup. This page is the root of the configuration reference: it covers the file format, the top-level keys, how the file is loaded, how to pull values from environment variables and from other parts of the file, and what a validation error looks like. It ends with a complete example. Each top-level section is documented on its own page, listed under [Root keys](#root-keys), which is the single source for that section's keys and defaults.

## Passing the config file

Pass the path with the `--config` flag. Kerberos will not start without it:

```bash
kerberos --config /etc/kerberos/config.json
```

```bash
docker run --rm \
  -p 30000:30000 -p 30001:30001 \
  -v "$PWD/config.json:/etc/kerberos/config.json:ro" \
  ghcr.io/trebent/kerberos:<version> --config /etc/kerberos/config.json
```

If you leave out `--config`, Kerberos prints `Error: --config flag is required` and the usage text, then exits with status 1. If the file cannot be read, startup fails with the operating system error, for example `open /etc/kerberos/config.json: no such file or directory`.

The config file controls what Kerberos does. Process-level settings come from environment variables instead: listening ports, server timeouts, logging, `OAS_DIRECTORY` and the OpenTelemetry exporter settings. See [Deployment: environment variables](./deployment.md#environment-variables) for the full list and their defaults.

## File format

- The file must be plain JSON. Comments, trailing commas, YAML and TOML are not supported.
- Only the `gateway` section is required. Every other top-level section is optional.
- Unknown keys are rejected in almost every object, so a misspelled key stops startup and does not get silently ignored. The `cookies` objects are the exception (see [Sessions: cookie configuration](./sessions.md#cookie-configuration)).
- Relative file paths in the config (TLS certificates and keys, OpenAPI specifications, the SQLite database file) are resolved against the working directory of the Kerberos process, not against the directory that holds the config file. Use absolute paths in containers.

The smallest working config has one backend:

```json
{
  "gateway": {
    "router": {
      "backends": [
        { "name": "echo", "host": "localhost", "port": 15000 }
      ]
    }
  }
}
```

With this file, Kerberos forwards `/gw/backend/echo/...` on the gateway port to `http://localhost:15000/...`. Every section you leave out takes the defaults documented on its own page. Two of them matter before you expose Kerberos: the Admin API always runs and creates a superuser with well-known default credentials (see [Admin API: the superuser](./admin-api.md#the-superuser)), and data is kept in a local SQLite file (see [Persistence](./persistence.md)).

## Root keys

This page owns the top level of the file. Each section is documented on one page only, the one in the **Documented in** column. That page is the single source for the section's keys, types, allowed values and defaults. Other pages link to it and do not repeat those values.

| Key | Required | What it configures | Documented in |
| --- | --- | --- | --- |
| `gateway` | yes | The backends behind the gateway (`router`) and HTTPS on the gateway port (`tls`). | [Routing](./routing.md) (`gateway.router`), [TLS](./tls.md) (`gateway.tls`) |
| `admin` | no | The first superuser (`superUser`) and the Admin API's cookies, origins and HTTPS (`api`). The Admin API runs whether or not this section is present. | [Admin API](./admin-api.md) |
| `auth` | no | Which backends need a login, which groups they require, and the basic-auth API. Leave it out to turn authentication off. | [Authentication](./authentication.md) |
| `oas` | no | Validating requests against each backend's OpenAPI specification. Leave it out to turn validation off. | [Request validation](./request-validation.md) |
| `observability` | no | Turning OpenTelemetry metrics and tracing on or off. | [Observability](./observability.md) |
| `persistence` | no | The database. | [Persistence](./persistence.md) |

### Shared objects

Some objects have the same shape wherever they appear. Each is documented once:

| Object | Appears in | Documented in |
| --- | --- | --- |
| `origins` | `gateway.router.backends[].origins`, `admin.api.origins`, `auth.methods.basic.api.origins` | [CORS: the `origins` object](./cors.md#the-origins-object) |
| `cookies` | `admin.api.cookies`, `auth.methods.basic.api.cookies` | [Sessions: cookie configuration](./sessions.md#cookie-configuration) |
| `tls` | `gateway.tls`, `admin.api.tls`, `gateway.router.backends[].tls` | [TLS](./tls.md) |
| `order` | `auth.order`, `oas.order` | [Routing: pipeline order](./routing.md#pipeline-order) |

Environment variables are not part of the file. They are documented in [Deployment: environment variables](./deployment.md#environment-variables).

## How the file is loaded

Loading happens once, at startup, in this order:

1. **Resolve references.** Every `${env:...}` and `${ref:...}` reference is replaced with its value. See [References](#references).
2. **Validate against the JSON schema.** The resolved document is checked against the schema files in [`internal/config/schemas/`](#schema-files). This step checks structure only: required keys, types, enums, ranges and unknown keys.
3. **Apply defaults.** Kerberos fills in values for settings you left out. Each section's page lists its defaults. Some schema files also contain `default` fields; those are descriptive only and are not applied to your file.

Any failure in steps 1 or 2 stops the process with exit status 1 before it opens any port. Other problems are found later in startup, after the config has loaded: a certificate, key or OpenAPI file that is missing or invalid, or a database that cannot be reached. These also stop the process.

Because references are resolved before validation, the schema checks the final values. A port that comes from an environment variable must still be a number in range.

## References

Any value in the file can be a reference that Kerberos replaces before validation:

| Syntax | Resolves to |
| --- | --- |
| `${env:NAME}` | The value of environment variable `NAME`. Startup fails if `NAME` is not set. |
| `${env:NAME:default}` | The value of `NAME`, or `default` if `NAME` is not set. |
| `${ref:path.to.field}` | The value of another field in the same file, addressed by its path from the root. |

### Environment references

```json
{
  "admin": {
    "superUser": {
      "clientId": "${env:KRB_SUPERUSER_ID:admin}",
      "clientSecret": "${env:KRB_SUPERUSER_SECRET}"
    }
  }
}
```

If `KRB_SUPERUSER_SECRET` is not set, startup fails with:

```text
environment variable not found: KRB_SUPERUSER_SECRET
```

The default applies only when the variable is **not set**. A variable that is set to an empty string resolves to an empty string, not to the default. `${env:NAME:}` (empty default) is allowed and resolves to an empty string when `NAME` is unset.

### Field references

A `${ref:...}` reference copies the value of another field. Separate object keys with `.`, and select array elements with `[n]` (counting from 0):

```json
{
  "gateway": {
    "router": {
      "backends": [
        { "name": "orders", "host": "orders.internal", "port": 8080 }
      ]
    }
  },
  "oas": {
    "mappings": [
      {
        "backend": "${ref:gateway.router.backends[0].name}",
        "specification": "/etc/kerberos/oas/orders.yaml"
      }
    ]
  }
}
```

Using a field reference for backend names means a renamed backend cannot get out of step with its `auth` and `oas` mappings.

If the referenced field itself holds a reference, Kerberos follows it. A field reference to a field that holds `${env:X}` resolves to the value of `X`.

If the path does not exist, startup fails with `referenced path was not found: <path>`.

### Quoted and unquoted references

A reference inside quotes always produces a JSON string. To produce a number or a boolean, write the reference **without** quotes:

```json
{
  "name": "orders",
  "host": "${env:ORDERS_HOST}",
  "port": ${env:ORDERS_PORT:8080},
  "timeout": ${env:ORDERS_TIMEOUT_MS:4000}
}
```

Unquoted references are inserted as raw text. The file is not valid JSON until Kerberos resolves them, so editors and tools such as `jq` will report errors in it.

| You write | Variable value | Result | Outcome |
| --- | --- | --- | --- |
| `"port": ${env:P}` | `8080` | `"port": 8080` | valid |
| `"port": "${env:P}"` | `8080` | `"port": "8080"` | schema error: `Invalid type. Expected: integer, given: string` |
| `"host": ${env:H}` | `orders.internal` | `"host": orders.internal` | JSON error: `invalid character 'o' looking for beginning of value` |
| `"port": ${env:P}` | (set but empty) | `"port": ` | JSON error: `invalid character '}' looking for beginning of value` |

Use quotes for strings and leave them off for numbers and booleans. The same rule applies to `${ref:...}`. If you reference a numeric field, leave the reference unquoted.

### Things to know about references

- **A reference must be the whole value.** You cannot build a value from a reference plus other text.
  - `"host": "orders-${env:ENV}"` fails at startup with a JSON error such as `invalid character '$' after object key:value pair`.
  - `"host": "${env:ENV}-orders"` does **not** fail. The text after the reference is silently dropped, so the result is just the value of `ENV`. Put the full value in the variable instead.
- **Restricted characters in names and defaults.** Variable names and default values can contain only letters, digits and `_`. A default that contains anything else, such as `.`, `/`, `-` or a space, fails with `malformed env var reference`. For example, `${env:DB_PATH:/data/krb.db}` and `${env:HOST:db.example.com}` both fail. Set such values through the variable itself and leave out the default.
- **A colon ends the default.** `${env:KRB_DB_ADDRESS:postgres:5432}` does not default to `postgres:5432`. The default becomes `postgres` and `:5432` is dropped without an error. You cannot give a `host:port` value as a default.
- **Field references have no default.** `${ref:path:default}` is treated as a path containing `:default` and fails with `referenced path was not found`.
- **Field references point at single values only.** A field reference can copy a string, number or boolean. A path to an object or an array, such as `${ref:gateway.router}`, fails with `referenced path was not found`.
- **Do not create circular references.** Keep reference chains short and acyclic.
- **Keep log verbosity low in production.** At `LOG_VERBOSITY` 100 or more, Kerberos logs every step of reference resolution. Use it only for local troubleshooting.
- **There is no way to write a literal `${env:...}` or `${ref:...}` string.** Any string value that starts with one of these prefixes is treated as a reference.

## Validation errors

When the resolved file breaks the schema, startup fails with one error that lists every problem as `<field path> - <description>`. Kerberos logs the error and then exits. With the default structured logging, the log line looks like this:

```json
{"level":"error","err":"schema validation failed: %!w(<nil>), gateway.router.backends.0.port - Invalid type. Expected: integer, given: string","name":"krb","msg":"Failed to set up configuration"}
```

Notes on reading these messages:

- Every message starts with `schema validation failed: %!w(<nil>), `. The `%!w(<nil>)` fragment is a formatting artefact and does not mean anything. The real problems follow it.
- Field paths use `.` for array indexes (`backends.0.port`), unlike `${ref:...}` paths, which use `[0]`. `(root)` means the top level of the file.
- Problems with an `origins` object (for example `allowAll` together with `allowedOrigins`) can show up as several lines, including `Must validate "then" as "if" was valid`. Look for the line that names a specific key.

Common messages:

| Message | Cause |
| --- | --- |
| `(root) - Additional property bogus is not allowed` | Unknown or misspelled key. The message names the key and the object it was found in. |
| `gateway.router - backends is required` | A required key is missing. |
| `gateway.router.backends.0.port - Must be greater than or equal to 1` | A value is out of range. |
| `admin.api.cookies.sameSite - admin.api.cookies.sameSite must be one of the following: "Strict", "Lax", "None"` | Enum values are case-sensitive: `Lax`, not `lax`. |
| `gateway.router.backends.0.origins.allowedOrigins - Array must have at most 0 items` | `allowAll` or `denyAll` combined with a non-empty `allowedOrigins`. |

Errors from the reference step and from JSON parsing are not prefixed with `schema validation failed`. They read like `environment variable not found: NAME`, `malformed env var reference: ...`, `referenced path was not found: ...` or a Go JSON message such as `invalid character '}' looking for beginning of object key string` (for example after a trailing comma).

## Complete example

This example uses every section: two backends (one over mTLS), HTTPS on both ports, basic authentication with group rules, OpenAPI validation for one backend and PostgreSQL. Secrets and environment-specific values come from environment variables.

```json
{
  "gateway": {
    "tls": {
      "serverCertFile": "/etc/kerberos/certs/gateway.pem",
      "serverKeyFile": "/etc/kerberos/certs/gateway.key"
    },
    "router": {
      "backends": [
        {
          "name": "orders",
          "host": "orders.internal",
          "port": 8080,
          "timeout": ${env:ORDERS_TIMEOUT_MS:4000},
          "origins": {
            "allowedOrigins": ["https://app.example.com"]
          }
        },
        {
          "name": "billing",
          "host": "billing.internal",
          "port": 8443,
          "tls": {
            "rootCAFile": "/etc/kerberos/certs/internal-ca.pem",
            "clientCertFile": "/etc/kerberos/certs/krb-client.pem",
            "clientKeyFile": "/etc/kerberos/certs/krb-client.key"
          }
        }
      ]
    }
  },
  "admin": {
    "superUser": {
      "clientId": "${env:KRB_SUPERUSER_ID:admin}",
      "clientSecret": "${env:KRB_SUPERUSER_SECRET}"
    },
    "api": {
      "cookies": {
        "sameSite": "Strict"
      },
      "origins": {
        "allowedOrigins": ["https://admin.example.com"]
      },
      "tls": {
        "serverCertFile": "/etc/kerberos/certs/admin.pem",
        "serverKeyFile": "/etc/kerberos/certs/admin.key"
      }
    }
  },
  "auth": {
    "methods": {
      "basic": {
        "api": {
          "cookies": {
            "domain": "example.com",
            "sameSite": "Lax"
          },
          "origins": {
            "allowedOrigins": ["https://app.example.com"]
          }
        }
      }
    },
    "scheme": {
      "mappings": [
        {
          "backend": "${ref:gateway.router.backends[0].name}",
          "method": "basic",
          "exempt": ["/public/*"],
          "authorization": {
            "groups": ["staff"],
            "paths": {
              "/refunds/*": ["finance"]
            }
          }
        },
        {
          "backend": "${ref:gateway.router.backends[1].name}",
          "method": "basic"
        }
      ]
    },
    "order": 1
  },
  "oas": {
    "mappings": [
      {
        "backend": "${ref:gateway.router.backends[0].name}",
        "specification": "/etc/kerberos/oas/orders.yaml",
        "options": {
          "validateBody": true
        }
      }
    ],
    "order": 100
  },
  "observability": {
    "enabled": true,
    "runtimeMetrics": true
  },
  "persistence": {
    "driver": "postgres",
    "address": "${env:KRB_DB_ADDRESS}",
    "postgres": {
      "database": "kerberos",
      "username": "${env:KRB_DB_USER}",
      "password": "${env:KRB_DB_PASSWORD}",
      "sslMode": "require"
    }
  }
}
```

Section by section:

- **`gateway.tls`** and **`admin.api.tls`**: both ports serve HTTPS only. The admin port also serves the basic-auth API. See [TLS](./tls.md).
- **`orders` backend**: reached at `/gw/backend/orders/...` (see [Routing](./routing.md) for naming rules). The timeout is an **unquoted** reference, so it resolves to a number. Keep it below the server write timeout (see [Deployment: timeouts](./deployment.md#timeouts)). Browser calls are allowed only from `https://app.example.com` (see [CORS](./cors.md)).
- **`billing` backend**: Kerberos connects over HTTPS, checks the backend's certificate against `internal-ca.pem` and presents its own client certificate (mTLS). See [TLS](./tls.md).
- **`admin.superUser`**: the secret has no fallback in its reference, so startup fails if `KRB_SUPERUSER_SECRET` is not set. See [Admin API: the superuser](./admin-api.md#the-superuser) for when these values are used.
- **`auth`**: both backends need a basic-auth login. On `orders`, paths matching `/public/*` need no login, every other path needs membership of `staff`, and `/refunds/*` needs `finance`. The `backend` values are field references, so they always match the router names. See [Authentication](./authentication.md).
- **`order`**: authentication (`1`) runs before validation (`100`). See [Routing: pipeline order](./routing.md#pipeline-order).
- **`oas`**: requests to `orders` are validated against its OpenAPI file, including bodies. See [Request validation](./request-validation.md).
- **`observability`**: exporters are configured with `OTEL_*` variables (see [Observability](./observability.md)).
- **`persistence`**: `KRB_DB_ADDRESS` must be `host:port`, for example `postgres.internal:5432`. It has no fallback because a reference default cannot contain `:` or `.` (see [Things to know about references](#things-to-know-about-references)). See [Persistence](./persistence.md).

To run it, set the variables the file needs:

```bash
export KRB_SUPERUSER_SECRET='change-me'
export KRB_DB_ADDRESS='postgres.internal:5432'
export KRB_DB_USER='kerberos'
export KRB_DB_PASSWORD='...'
kerberos --config /etc/kerberos/config.json
```

## Things to know

- **Keep backend names in step.** Backend names in `auth.scheme.mappings` and `oas.mappings` must match a router backend exactly. Use `${ref:gateway.router.backends[n].name}` for these names, and check the flow with the admin API after changes (see [Admin API](./admin-api.md)).
- **Some effective defaults differ from the schema files.** The `default` fields in the schemas are not applied. Rely on the section pages, and write values such as `order` and `validateBody` out explicitly.
- **The config is read only at startup.** Restart Kerberos to apply changes.

## Schema files

The JSON Schema (draft-07) files under `internal/config/schemas/` are the authoritative reference for every key, type, enum, range and required field. Kerberos embeds them in the binary at build time.

| File | Covers | Documented in |
| --- | --- | --- |
| [`config_schema.json`](../internal/config/schemas/config_schema.json) | Top level | This page |
| [`gateway_schema.json`](../internal/config/schemas/gateway_schema.json) | `gateway`, including `gateway.tls` | [Routing](./routing.md), [TLS](./tls.md) |
| [`router_schema.json`](../internal/config/schemas/router_schema.json) | `gateway.router.backends[]`, including backend `tls` | [Routing](./routing.md), [TLS](./tls.md) |
| [`admin_schema.json`](../internal/config/schemas/admin_schema.json) | `admin` | [Admin API](./admin-api.md) |
| [`auth_schema.json`](../internal/config/schemas/auth_schema.json) | `auth` | [Authentication](./authentication.md) |
| [`oas_schema.json`](../internal/config/schemas/oas_schema.json) | `oas` | [Request validation](./request-validation.md) |
| [`observability_schema.json`](../internal/config/schemas/observability_schema.json) | `observability` | [Observability](./observability.md) |
| [`persistence_schema.json`](../internal/config/schemas/persistence_schema.json) | `persistence` | [Persistence](./persistence.md) |
| [`cookies_schema.json`](../internal/config/schemas/cookies_schema.json) | Shared `cookies` object | [Sessions](./sessions.md#cookie-configuration) |
| [`origins_schema.json`](../internal/config/schemas/origins_schema.json) | Shared `origins` object | [CORS](./cors.md#the-origins-object) |
| [`ordered_schema.json`](../internal/config/schemas/ordered_schema.json) | Shared `order` value | [Routing](./routing.md#pipeline-order) |

The admin-connector uses a separate config file with its own schema, [`connector_schema.json`](../internal/config/schemas/connector_schema.json). The same reference syntax works there. See [admin-connector](./admin-connector.md).

## Related docs

- [Deployment](./deployment.md): CLI flags, environment variables, ports and timeouts
- [Routing](./routing.md), [TLS](./tls.md), [CORS](./cors.md)
- [Authentication](./authentication.md), [Sessions](./sessions.md), [Organisations](./organizations.md)
- [Request validation](./request-validation.md)
- [Admin API](./admin-api.md)
- [Observability](./observability.md)
- [Persistence](./persistence.md)
