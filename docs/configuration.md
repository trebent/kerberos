# Configuration

Kerberos is configured with a single JSON file passed via the `--config` flag. The file is loaded, reference-resolved, schema-validated, and then unmarshalled into the `RootConfig` struct.

---

## Reference System

Config values can reference environment variables or other values within the same config file using `${...}` syntax.

*NOTE:* quoting a referenced value means the resulting value is quoted. For referencing numerical values, omit quotes to ensure the schema validation does not throw a validation error due to incorrect type usage.

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

Resolves to the value at the given JSON path within the *same* config file. The path uses dot-notation for object keys and bracket-notation for array indices (e.g., `gateway.router.backends[0].name`).

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

### `observability` (optional)

Controls OpenTelemetry tracing and metrics. Defaults to enabled.

```json
"observability": {
  "enabled": true,
  "runtimeMetrics": true
}
```

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
