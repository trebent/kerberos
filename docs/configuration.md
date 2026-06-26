# Configuration

Kerberos is configured with a single JSON file passed via the `--config` flag. The file is loaded, reference-resolved, schema-validated, and then unmarshalled into the `RootConfig` struct.

---

## Reference System

Config values can reference environment variables or other values within the same config file using `${...}` syntax.

### Environment Variable References

```json
"host": "${env:BACKEND_HOST}"
```

Resolves to the value of the `BACKEND_HOST` environment variable. Returns an error at startup if the variable is not set.

```json
"host": "${env:BACKEND_HOST:localhost}"
```

Resolves to `BACKEND_HOST` if set, otherwise falls back to `localhost`.

### Path References

```json
"host": "${ref:gateway.router.backends[0].host}"
```

Resolves to the value at the given JSON path within the _same_ config file. The path uses dot-notation for object keys and bracket-notation for array indices (e.g., `gateway.router.backends[0].name`).

Path references support nesting: a value pointed to by a `${ref:...}` can itself be a `${ref:...}` or `${env:...}`, and the resolver will follow the chain. Circular references are detected and cause a startup error.

References are resolved before schema validation, so schemas always see the final concrete values.

---

## Schema Validation

The parsed config is validated against JSON Schema (Draft 7) before unmarshalling. Schemas live in `internal/config/schemas/` and are embedded into the binary at build time.

The root schema (`config_schema.json`) uses `$ref` to reference sub-schemas by their `$id` URL (e.g., `http://trebent.com/kerberos/schemas/gateway_schema.json`). Each section's schema is registered separately so they can also reference each other.

All object schemas use `additionalProperties: false`, which means unknown fields cause a validation error at startup.

---

## Config Sections

### `gateway` (required)

Controls how the API gateway accepts connections and which backends it can route to.

```json
"gateway": {
  "router": {
    "backends": [
      {
        "name": "my-service",
        "host": "localhost",
        "port": 8080,
        "timeout": 5000,
        "tls": {
          "rootCAFile": "/certs/ca.pem",
          "clientCertFile": "/certs/client.pem",
          "clientKeyFile": "/certs/client-key.pem",
          "insecureSkipVerify": false
        }
      }
    ]
  },
  "tls": {
    "serverCertFile": "/certs/server.pem",
    "serverKeyFile": "/certs/server-key.pem"
  }
}
```

| Field | Type | Required | Default | Notes |
|---|---|---|---|---|
| `router.backends[].name` | string | yes | — | Must be unique. Used in URL routing: `/gw/backend/<name>/...` |
| `router.backends[].host` | string | yes | — | Hostname or IP of the backend |
| `router.backends[].port` | integer | yes | — | Port of the backend (1–65535) |
| `router.backends[].timeout` | integer | no | `5000` | Request timeout in milliseconds |
| `router.backends[].tls` | object | no | — | Omit for plain HTTP. `rootCAFile` is required when using TLS |
| `router.backends[].tls.rootCAFile` | string | yes (if tls) | — | PEM CA bundle to verify the backend certificate |
| `router.backends[].tls.clientCertFile` | string | no | — | PEM client certificate for mTLS. Must be paired with `clientKeyFile` |
| `router.backends[].tls.clientKeyFile` | string | no | — | PEM client key for mTLS. Must be paired with `clientCertFile` |
| `router.backends[].tls.insecureSkipVerify` | boolean | no | `false` | Disables certificate verification. Non-production use only |
| `tls.serverCertFile` | string | yes (if tls) | — | PEM certificate for the gateway's own TLS identity |
| `tls.serverKeyFile` | string | yes (if tls) | — | PEM key for the gateway's own TLS identity |

### `observability` (optional)

Controls OpenTelemetry tracing and metrics. Defaults to enabled.

```json
"observability": {
  "enabled": true,
  "runtimeMetrics": true
}
```

| Field | Type | Required | Default | Notes |
|---|---|---|---|---|
| `enabled` | boolean | no | `true` | Enables OTEL tracing and metrics. When `false`, a lightweight logging-only path is used |
| `runtimeMetrics` | boolean | no | `true` | Enables Go runtime metrics collection via OTEL |

### `admin` (optional)

Controls the admin API server and its superuser credentials. Defaults are applied when this section is omitted.

```json
"admin": {
  "superUser": {
    "clientId": "admin",
    "clientSecret": "secret"
  },
  "api": {
    "tls": {
      "serverCertFile": "/certs/admin-server.pem",
      "serverKeyFile": "/certs/admin-server-key.pem"
    }
  }
}
```

| Field | Type | Required | Default | Notes |
|---|---|---|---|---|
| `superUser.clientId` | string | no | `"admin"` | Superuser login name |
| `superUser.clientSecret` | string | no | `"secret"` | Superuser password. Change in production |
| `api.tls` | object | no | — | Enables TLS on the admin server. Omit for plain HTTP |

### `auth` (optional)

Enables authentication and authorisation for backend routes. When present, `methods` and `scheme` must both be provided.

The `order` field controls where the Auth flow component runs within the custom block relative to other ordered components (e.g., the OAS validator). Lower values run first.

```json
"auth": {
  "order": 1,
  "methods": {
    "basic": {}
  },
  "scheme": {
    "mappings": [
      {
        "backend": "my-service",
        "method": "basic",
        "exempt": ["/health"],
        "authorization": {
          "groups": ["users"],
          "paths": {
            "/admin/*": ["admins"]
          }
        }
      }
    ]
  }
}
```

| Field | Type | Required | Default | Notes |
|---|---|---|---|---|
| `order` | integer | no | `1` | Execution order within the custom block. Minimum `1` |
| `methods.basic` | object | no | — | Enables basic (session-cookie) authentication |
| `scheme.mappings[].backend` | string | yes | — | Backend name this mapping applies to |
| `scheme.mappings[].method` | string | yes | — | Auth method. Currently only `"basic"` is supported |
| `scheme.mappings[].exempt` | array of strings | no | — | Paths exempted from authentication. Matched using [`path.Match`](https://pkg.go.dev/path#Match) |
| `scheme.mappings[].authorization.groups` | array of strings | no | — | Base groups required for any path. A user in _any_ listed group is permitted |
| `scheme.mappings[].authorization.paths` | object | no | — | Per-path group overrides. Key is path pattern, value is array of permitted groups |

### `oas` (optional)

Enables OpenAPI Specification validation for incoming requests to mapped backends. The `order` field controls where the OAS validator runs within the custom block.

```json
"oas": {
  "order": 2,
  "mappings": [
    {
      "backend": "my-service",
      "specification": "/oas/my-service.yaml",
      "options": {
        "validateBody": true
      }
    }
  ]
}
```

| Field | Type | Required | Default | Notes |
|---|---|---|---|---|
| `order` | integer | no | `1` | Execution order within the custom block. Minimum `1` |
| `mappings[].backend` | string | yes | — | Backend name to validate |
| `mappings[].specification` | string | yes | — | Path to the OAS file (YAML or JSON) |
| `mappings[].options.validateBody` | boolean | no | `true` | When `false`, request bodies are not validated |

### `persistence` (optional)

Selects the backing database for admin data (users, sessions, groups). Defaults to SQLite.

```json
"persistence": {
  "driver": "sqlite",
  "address": "krb.db"
}
```

For PostgreSQL:

```json
"persistence": {
  "driver": "postgres",
  "address": "localhost:5432",
  "postgres": {
    "database": "kerberos",
    "username": "krb",
    "password": "${env:DB_PASSWORD}",
    "sslMode": "require"
  }
}
```

| Field | Type | Required | Default | Notes |
|---|---|---|---|---|
| `driver` | string | yes | `"sqlite"` | `"sqlite"` or `"postgres"` |
| `address` | string | yes | `"krb.db"` | File path for SQLite, `host:port` for PostgreSQL |
| `postgres.database` | string | yes (postgres) | — | PostgreSQL database name |
| `postgres.username` | string | no | — | PostgreSQL user |
| `postgres.password` | string | no | — | PostgreSQL password |
| `postgres.sslMode` | string | no | — | PostgreSQL SSL mode (e.g., `"disable"`, `"require"`, `"verify-full"`) |

---

## Annotated Full Example

This example shows all config sections together with references in use.

```json
{
  "gateway": {
    "router": {
      "backends": [
        {
          "name": "api",
          "host": "${env:API_HOST:localhost}",
          "port": 8080,
          "timeout": 3000
        },
        {
          "name": "secure-api",
          "host": "secure.internal",
          "port": 8443,
          "tls": {
            "rootCAFile": "/certs/internal-ca.pem"
          }
        }
      ]
    },
    "tls": {
      "serverCertFile": "/certs/krb.pem",
      "serverKeyFile": "/certs/krb-key.pem"
    }
  },

  "observability": {
    "enabled": true,
    "runtimeMetrics": false
  },

  "admin": {
    "superUser": {
      "clientId": "${env:ADMIN_USER:admin}",
      "clientSecret": "${env:ADMIN_SECRET}"
    }
  },

  "auth": {
    "order": 1,
    "methods": {
      "basic": {}
    },
    "scheme": {
      "mappings": [
        {
          "backend": "api",
          "method": "basic",
          "exempt": ["/api/health", "/api/version"],
          "authorization": {
            "groups": ["users"],
            "paths": {
              "/api/admin/*": ["admins"]
            }
          }
        }
      ]
    }
  },

  "oas": {
    "order": 2,
    "mappings": [
      {
        "backend": "api",
        "specification": "/oas/api.yaml",
        "options": {
          "validateBody": true
        }
      }
    ]
  },

  "persistence": {
    "driver": "postgres",
    "address": "${env:DB_HOST:localhost}:5432",
    "postgres": {
      "database": "kerberos",
      "username": "${env:DB_USER}",
      "password": "${env:DB_PASSWORD}",
      "sslMode": "require"
    }
  }
}
```

Key points illustrated:

- `${env:API_HOST:localhost}` — uses the env var with a fallback.
- `${env:ADMIN_SECRET}` — requires the env var to be set (no fallback).
- `auth.order: 1`, `oas.order: 2` — auth runs before OAS validation within the custom block.
- `auth` exempt paths use `path.Match` glob patterns.
- `persistence.address` composes an env var into a longer string value.
