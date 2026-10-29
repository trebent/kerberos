# Configuration

Kerberos reads its settings from one JSON file that you name at startup. This page covers how that file is loaded, how to pull values from environment variables and from other parts of the file, the defaults Kerberos uses when you leave something out, and what a validation error looks like. It ends with a complete example. Each section of the file has its own page, linked from the [section overview](#sections-at-a-glance), which covers that feature in detail.

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

The config file controls what Kerberos does. Process-level settings come from environment variables instead: listening ports (`PORT`, default 30000; `ADMIN_PORT`, default 30001), server timeouts, logging, `OAS_DIRECTORY` and the OpenTelemetry exporter settings. See [Deployment: environment variables](./deployment.md#environment-variables) for the full list.

## File format

- The file must be plain JSON. Comments, trailing commas, YAML and TOML are not supported.
- Only the `gateway` section is required. Every other top-level section is optional.
- Unknown keys are rejected in almost every object, so a misspelled key stops startup and does not get silently ignored. The `cookies` objects are the exception (see [Things to know](#things-to-know)).
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

With this file, Kerberos forwards `http://localhost:30000/gw/backend/echo/...` to `http://localhost:15000/...`, keeps its data in a SQLite file named `krb.db` in the working directory, and enables the admin API on port 30001 with the default superuser `admin` / `secret`. Change that password before you expose Kerberos (see [Admin API](./admin-api.md)).

## How the file is loaded

Loading happens once, at startup, in this order:

1. **Resolve references.** Every `${env:...}` and `${ref:...}` reference is replaced with its value. See [References](#references).
2. **Validate against the JSON schema.** The resolved document is checked against the schema files in [`internal/config/schemas/`](#schema-files). This step checks structure only: required keys, types, enums, ranges and unknown keys.
3. **Apply defaults.** Kerberos fills in values for settings you left out. See [Defaults](#defaults).

Any failure in steps 1 or 2 stops the process with exit status 1 before it opens any port. Other problems are found later in startup, after the config has loaded: a certificate, key or OpenAPI file that is missing or invalid, or a database that cannot be reached. These also stop the process. Some mistakes are never reported. For example, an `auth` or `oas` mapping can name a backend that does not exist, and Kerberos starts anyway (see [Things to know](#things-to-know)).

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
  "host": "${env:ORDERS_HOST:orders.internal}",
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
- **Values are inserted verbatim.** Kerberos does not JSON-escape resolved values. A value that contains `"` breaks the file, and a `\` is read as the start of a JSON escape sequence. This matters most for passwords. Avoid these characters, or escape them in the value yourself (for example `a\"b`).
- **Field references have no default.** `${ref:path:default}` is treated as a path containing `:default` and fails with `referenced path was not found`.
- **Field references point at single values only.** A field reference can copy a string, number or boolean. A path to an object or an array, such as `${ref:gateway.router}`, fails with `referenced path was not found`.
- **Do not create circular references.** A field that refers to itself, directly or through other fields, normally fails with `cischular reference detected: <path>` (the spelling is as printed). Some reference chains that lead into a loop are not detected, and startup can then hang. Keep reference chains short and acyclic.
- **High log verbosity prints resolved secrets.** With `LOG_VERBOSITY` set to 100 or more, Kerberos logs every step of reference resolution, including the fully resolved config file. Secrets that came from environment variables then appear in the logs. Do not use that verbosity in production.
- **There is no way to write a literal `${env:...}` or `${ref:...}` string.** Any string value that starts with one of these prefixes is treated as a reference.

## Validation errors

When the resolved file breaks the schema, startup fails with one error that lists every problem as `<field path> - <description>`. Kerberos logs the error and then exits. With the default structured logging, the log line looks like this:

```json
{"level":"error","err":"schema validation failed: %!w(<nil>), gateway.router.backends.0.port - Invalid type. Expected: integer, given: string","name":"krb","msg":"Failed to set up configuration"}
```

Notes on reading these messages:

- Every message starts with `schema validation failed: %!w(<nil>), `. The `%!w(<nil>)` fragment is a formatting artifact and does not mean anything. The real problems follow it.
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

## Sections at a glance

| Section | Required | What it configures | Details |
| --- | --- | --- | --- |
| `gateway` | yes | Backends behind the gateway (`router.backends`: name, host, port, timeout, per-backend CORS origins and TLS to the backend) and optional HTTPS for the gateway port (`tls`). | [Routing](./routing.md), [TLS](./tls.md), [CORS](./cors.md) |
| `admin` | no | First superuser credentials (`superUser`) and the admin API's cookies, CORS origins and HTTPS (`api`). The admin API is always enabled. | [Admin API](./admin-api.md), [Sessions](./sessions.md), [TLS](./tls.md) |
| `auth` | no | Access control for backends: the basic-auth method and its API (`methods.basic`), which backends need a login and which groups (`scheme.mappings`), and the pipeline position (`order`). Leave it out to turn authentication off. | [Authentication](./authentication.md), [Sessions](./sessions.md), [Organisations](./organizations.md) |
| `oas` | no | OpenAPI request validation per backend (`mappings`: backend, specification file, `validateBody`) and the pipeline position (`order`). Leave it out to turn validation off. | [Request validation](./request-validation.md) |
| `observability` | no | Turns OpenTelemetry metrics and tracing (`enabled`) and Go runtime metrics (`runtimeMetrics`) on or off. Exporters are set with `OTEL_*` environment variables. | [Observability](./observability.md) |
| `persistence` | no | The database: `driver` (`sqlite` or `postgres`), `address`, and PostgreSQL settings (`postgres`). | [Persistence](./persistence.md) |

## Shared building blocks

Some objects have the same shape wherever they appear.

### `cookies`

Used in `admin.api.cookies` and `auth.methods.basic.api.cookies`. Controls the attributes of the session, refresh and CSRF cookies set by that API.

| Key | Type | Default | Description |
| --- | --- | --- | --- |
| `domain` | string | not set (host-only cookies) | Becomes `Domain=<value>` on the cookies. |
| `sameSite` | `"Strict"`, `"Lax"` or `"None"` | `"Strict"` | Becomes `SameSite=<value>` on the cookies. |

See [Sessions: cookie configuration](./sessions.md#cookie-configuration).

### `origins`

Used in `gateway.router.backends[].origins`, `admin.api.origins` and `auth.methods.basic.api.origins`. Controls which browser origins may call that backend or API.

| Key | Type | Description |
| --- | --- | --- |
| `allowedOrigins` | array of strings | Exact origins (`scheme://host[:port]`) that are allowed. |
| `allowAll` | boolean | Allow any origin. |
| `denyAll` | boolean | Reject every request that carries an `Origin` header. |

Use only one of the three options. If you leave out `origins`, or set none of the three options, Kerberos does no CORS handling for that backend or API. See [CORS](./cors.md).

### `order`

Used in `auth.order` and `oas.order`. A whole number of 1 or more that positions authentication and request validation in the request pipeline. Lower numbers run first. Set it explicitly on both sections and give them different values. The usual choice is `auth` `1` and `oas` `100`, so unauthenticated requests are rejected before their bodies are validated. If you leave `order` out, it becomes `0` (not the schema's `1`), and with equal values the order between the two is not guaranteed.

## Defaults

These are the values Kerberos actually uses when you leave a setting out. Some schema files also contain `default` fields. Those are documentation only and are not applied to your file. Where they differ from the real behaviour, the table says so.

| Setting | Default when left out | Notes |
| --- | --- | --- |
| `gateway.tls` | not set: the gateway port serves plain HTTP | |
| `gateway.router.backends[].timeout` | `5000` (ms) | The `WRITE_TIMEOUT_SECONDS` environment variable (default 5 s) also limits each gateway response. Raise both for slow backends. See [Deployment: timeouts](./deployment.md#timeouts). |
| `gateway.router.backends[].origins` | not set: no CORS handling | |
| `gateway.router.backends[].tls` | not set: plain HTTP to the backend | With a `tls` block, `rootCAFile` is required. An empty string uses the system CA pool. `insecureSkipVerify` defaults to `false`. |
| `admin.superUser` | `clientId` `admin`, `clientSecret` `secret` | Used only on the first start against an empty database. Changing it later has no effect. See [Admin API](./admin-api.md). |
| `admin.api.cookies` | `sameSite` `Strict`, no `domain` | |
| `admin.api.origins` | no CORS handling | |
| `admin.api.tls` | not set: the admin port serves plain HTTP | |
| `auth` | not set: no authentication on any backend, and the basic-auth API is not served | |
| `auth.methods.basic.api.cookies` | `sameSite` `Strict`, no `domain` | |
| `auth.methods.basic.api.origins` | no CORS handling | |
| `auth.order` | `0` | The schema says `1`. Always set it. |
| `oas` | not set: no request validation | |
| `oas.order` | `0` | The schema says `1`. Always set it. |
| `oas.mappings[].options` | `validateBody: true` | Only when `options` is left out entirely. `"options": {}` turns body validation **off**, although the schema says `true`. Always write `validateBody` explicitly. |
| `observability.enabled` | `true` | |
| `observability.runtimeMetrics` | `true` | Settings you leave out of a partial block keep their defaults. `{"enabled": false}` leaves `runtimeMetrics` at `true`, which has no effect while observability is off. |
| `persistence` | `driver` `sqlite`, `address` `krb.db` | `krb.db` is relative to the working directory. If you include a `persistence` block, both `driver` and `address` are required. |
| `persistence.postgres.sslMode` | not sent: the PostgreSQL driver's default (`require`) applies | See [Persistence](./persistence.md). |

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

- **`gateway.tls`**: the gateway port (30000) serves HTTPS only. See [TLS](./tls.md).
- **`orders` backend**: reached at `/gw/backend/orders/...`. Use lowercase letters, digits, `-` and `_` in backend names, because other names cannot be reached (see [Routing](./routing.md)). The timeout is an **unquoted** reference, so it resolves to a number. It defaults to 4000 ms, which stays under the default 5 s server write timeout. Browser calls are allowed only from `https://app.example.com`.
- **`billing` backend**: Kerberos connects over HTTPS, checks the backend's certificate against `internal-ca.pem` and presents its own client certificate (mTLS). With no `origins` block, Kerberos does no CORS handling for this backend.
- **`admin.superUser`**: bootstraps the first superuser. The secret has no default, so startup fails if `KRB_SUPERUSER_SECRET` is not set. These values are used only when the database has no superuser yet.
- **`admin.api`**: the admin port (30001) serves HTTPS only. That includes the basic-auth API, which shares the port. Only `https://admin.example.com` may call it from a browser.
- **`auth`**: both backends need a basic-auth login. On `orders`, paths matching `/public/*` need no login, every other path needs membership of `staff`, and `/refunds/*` needs `finance`. See [Authentication](./authentication.md) for how paths and groups are matched. The `backend` values are field references, so they always match the router names. `order: 1` runs authentication before validation.
- **`oas`**: requests to `orders` are validated against its OpenAPI file, including bodies. `validateBody` is written out because `"options": {}` would turn body validation off. See [Request validation](./request-validation.md).
- **`observability`**: the defaults, written out for clarity. Exporters are configured with `OTEL_*` variables (see [Observability](./observability.md)).
- **`persistence`**: `KRB_DB_ADDRESS` must be `host:port`, for example `postgres.internal:5432`. It has no default because a default value cannot contain `:` or `.`. See [Persistence](./persistence.md).

To run it, set the variables the file needs:

```bash
export KRB_SUPERUSER_SECRET='change-me'
export KRB_DB_ADDRESS='postgres.internal:5432'
export KRB_DB_USER='kerberos'
export KRB_DB_PASSWORD='...'
kerberos --config /etc/kerberos/config.json
```

## Things to know

- **Cross-references between sections are not checked.** A backend name in `auth.scheme.mappings` or `oas.mappings` that does not match any router backend is accepted, and Kerberos starts. A misspelled name in an `auth` mapping leaves the intended backend **without authentication**. Use `${ref:gateway.router.backends[n].name}` for these names, and check the flow with the admin API after changes (see [Admin API](./admin-api.md)).
- **Some incomplete sections crash startup.** A `gateway` without `router`, and an `auth` section with neither `methods` nor `scheme` (for example `"auth": {"order": 1}`), both pass schema validation but make Kerberos exit with a Go panic (`nil pointer dereference`) instead of a clear message. Always include `gateway.router.backends`, and include both `methods` and `scheme` when you add `auth`.
- **`cookies` objects accept unknown keys.** A misspelled key such as `"samesite"` (lower-case `s`) inside `cookies` is ignored without an error, and the default `Strict` stays in effect.
- **`order` and `options` defaults differ from the schema.** See [Defaults](#defaults). Write `order` and `validateBody` out explicitly.
- **The superuser block is read only once.** After the first start, change the superuser password through the admin API. Editing the config has no effect.
- **The config is read only at startup.** Restart Kerberos to apply changes.

## Schema files

The JSON Schema (draft-07) files under `internal/config/schemas/` are the authoritative reference for every key, type, enum, range and required field. Kerberos embeds them in the binary at build time.

| File | Covers |
| --- | --- |
| [`config_schema.json`](../internal/config/schemas/config_schema.json) | Top level: the six sections, `gateway` required |
| [`gateway_schema.json`](../internal/config/schemas/gateway_schema.json) | `gateway`, including `gateway.tls` |
| [`router_schema.json`](../internal/config/schemas/router_schema.json) | `gateway.router.backends[]`, including backend `tls` |
| [`admin_schema.json`](../internal/config/schemas/admin_schema.json) | `admin` |
| [`auth_schema.json`](../internal/config/schemas/auth_schema.json) | `auth` |
| [`oas_schema.json`](../internal/config/schemas/oas_schema.json) | `oas` |
| [`observability_schema.json`](../internal/config/schemas/observability_schema.json) | `observability` |
| [`persistence_schema.json`](../internal/config/schemas/persistence_schema.json) | `persistence` |
| [`cookies_schema.json`](../internal/config/schemas/cookies_schema.json) | Shared `cookies` object |
| [`origins_schema.json`](../internal/config/schemas/origins_schema.json) | Shared `origins` object |
| [`ordered_schema.json`](../internal/config/schemas/ordered_schema.json) | Shared `order` value |

The `default` values in these files are descriptive only. The [Defaults](#defaults) table lists the values that take effect.

The admin-connector uses a separate config file with its own schema, [`connector_schema.json`](../internal/config/schemas/connector_schema.json). The same reference syntax works there. See [admin-connector](./admin-connector.md).

## Related docs

- [Deployment](./deployment.md): CLI flags, environment variables, ports and timeouts
- [Routing](./routing.md), [TLS](./tls.md), [CORS](./cors.md)
- [Authentication](./authentication.md), [Sessions](./sessions.md), [Organisations](./organizations.md)
- [Request validation](./request-validation.md)
- [Admin API](./admin-api.md)
- [Observability](./observability.md)
- [Persistence](./persistence.md)
