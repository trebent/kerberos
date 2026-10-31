# Request validation

Kerberos can check each incoming request against an OpenAPI specification for its backend before forwarding it. Requests that do not match the specification, such as an unknown path, an unsupported method, a missing parameter or a malformed JSON body, are rejected at the gateway with a JSON error and never reach your service.

## Why use it

- **Protect backends from malformed input.** Your service only receives requests that match its published contract.
- **Expose only what you publish.** Paths and methods that are not in the specification return `404` at the gateway, even if the backend has them, for example internal or debug endpoints.
- **One contract, one place.** The same specification you publish to API consumers is enforced at the edge. Operators can read the active specification through the [Admin API](./admin-api.md).

Validation is opt-in per backend. Backends without a mapping are forwarded without any validation.

## How it works

1. The router matches `/gw/backend/<name>/...` to a backend and strips the `/gw/backend/<name>` prefix (see [Routing](./routing.md)).
2. If the backend has an OpenAPI mapping, Kerberos looks up the operation for the remaining path and method in that backend's specification.
3. Kerberos validates the request against that operation. If validation fails, it responds with an error and stops. If validation passes, the request moves on to the next step in the pipeline and then to the backend.

Validation runs alongside [authentication](./authentication.md). The `order` values of the `auth` and `oas` sections decide which runs first (see [Ordering with authentication](#ordering-with-authentication)).

Specifications are loaded once, at startup. To change a specification, edit the file and restart Kerberos.

## Configuration

Add an `oas` section to the configuration file:

```json
{
  "gateway": {
    "router": {
      "backends": [
        { "name": "echo", "host": "localhost", "port": 15000 },
        { "name": "orders", "host": "orders.internal", "port": 8080 }
      ]
    }
  },
  "oas": {
    "mappings": [
      {
        "backend": "echo",
        "specification": "test/oas/echo.yaml",
        "options": { "validateBody": true }
      },
      {
        "backend": "orders",
        "specification": "/etc/kerberos/oas/orders.yaml",
        "options": { "validateBody": false }
      }
    ],
    "order": 100
  }
}
```

You can use a [config reference](./configuration.md) such as `"${ref:gateway.router.backends[0].name}"` for `backend` so that the mapping stays in sync with the backend name. The example configurations in `test/config/` do this.

| Key | Type | Default | Description |
|---|---|---|---|
| `oas.mappings` | array | (required) | One entry per backend to validate. At least one entry is required. |
| `oas.mappings[].backend` | string | (required) | Name of a backend from `gateway.router.backends`. Must match exactly. |
| `oas.mappings[].specification` | string | (required) | Path to the OpenAPI file (YAML or JSON). Relative paths are resolved against the working directory of the Kerberos process. |
| `oas.mappings[].options` | object | `{ "validateBody": true }` when omitted | Validation options for this backend. Always set `validateBody` when you include `options`. |
| `oas.mappings[].options.validateBody` | boolean | `true` when `options` is omitted, `false` when `options` is set without it | When `true`, request bodies are checked against the operation's `requestBody`. When `false`, any body, or no body, is accepted. Paths, methods and parameters are still checked. |
| `oas.order` | integer | see [Routing: pipeline order](./routing.md#pipeline-order) | Position of the validator relative to [authentication](./authentication.md). Set it explicitly. |

Leave the whole `oas` section out to turn request validation off. Unknown keys are rejected at startup. The authoritative schema is [`internal/config/schemas/oas_schema.json`](../internal/config/schemas/oas_schema.json).

## Writing the specification

### Paths are backend-relative

Write paths as your backend sees them, not as clients see them on the gateway. Kerberos strips `/gw/backend/<name>` before it validates the request.

| Client request | Path that is validated and forwarded |
|---|---|
| `GET /gw/backend/echo/hi` | `GET /hi` |
| `GET /gw/backend/echo/long/hello` | `GET /long/hello` |
| `GET /gw/backend/echo/` | `GET /` |

So a specification for the `echo` backend declares `/hi`, not `/gw/backend/echo/hi`:

```yaml
openapi: 3.0.4
info:
  title: Echo
  version: 0.1.0
paths:
  /hi:
    post:
      operationId: sayHiPost
      requestBody:
        required: true
        content:
          application/json:
            schema:
              type: object
              properties:
                name:
                  type: string
      responses:
        "200":
          description: Echo response.
```

See `test/oas/echo.yaml` in the repository for a complete example.

### The `servers` section

Kerberos checks paths only, so the `servers` list does not affect validation. Keep whatever `servers` entries suit your API consumers. There is no need to add the `/gw/backend/<name>` prefix there.

### OpenAPI version

Use OpenAPI 3.0.x. All specifications in the repository use `openapi: 3.0.4`.

Keep the specification in one file: put shared schemas under `components` and refer to them with local `$ref`s (`#/components/...`).

### Access control

Request validation checks the shape of a request. To control who may call a backend, use the gateway's [authentication](./authentication.md). `securitySchemes` and `security` entries in the specification do not affect validation.

## What is validated

For a backend with a mapping, every request is checked for:

| Check | Notes |
|---|---|
| Path | Must match a path in `paths`, including templated paths such as `/items/{id}`. |
| Method | The operation must exist for that path. |
| Path parameters | Type, format and schema constraints. |
| Query parameters | Required parameters and schema constraints. |
| Header and cookie parameters | When declared as `in: header` or `in: cookie` parameters. |
| Request body | Only when `validateBody` is `true`: required-ness, `Content-Type` against the declared `content` types, and the body against the schema. |

Only requests are checked, not the backend's responses.

CORS preflight requests (`OPTIONS` with an `Origin` header) to a backend with an `origins` configuration are answered by the router before validation, so the specification does not need `options` operations for them. See [CORS](./cors.md).

## Responses and errors

When validation fails, Kerberos responds with `Content-Type: application/json` and a body in the gateway's standard error format:

```json
{ "errors": ["<message>"] }
```

| Situation | Status | Example message |
|---|---|---|
| Path not in the specification | `404` | `no matching operation was found` |
| Path exists but the method does not | `404` | `method not allowed` |
| Parameter or body fails validation | `400` | `request body has an error: value is required but missing` |
| Any other validation failure | `500` | (library message) |

The message text is meant for people to read. Do not parse it in code, because it can change between versions. Messages do not include your schema, so callers do not see your specification.

Example using the echo backend from `test/config/local.json` (that configuration also enables basic authentication on `echo`, so add your session cookies or remove the `auth` section to try this):

```bash
# Unknown path -> 404
curl -i -X GET http://localhost:30000/gw/backend/echo/does-not-exist

# POST /hi requires a JSON body -> 400
curl -i -X POST http://localhost:30000/gw/backend/echo/hi

# Valid request -> forwarded to the backend
curl -i -X POST http://localhost:30000/gw/backend/echo/hi \
  -H 'Content-Type: application/json' \
  -d '{"name": "kerberos"}'
```

Failed validations are also recorded in [debug sessions](./admin-debugging.md) as a failed `oas-validator` step.

## Ordering with authentication

Validation and authentication are both ordered steps that run after routing and before forwarding, in ascending `order`. Run authentication first, so that callers who are not logged in learn nothing about your API from validation errors. See [Routing: pipeline order](./routing.md#pipeline-order) for how `order` works and its default.

## Viewing the active specification

Operators can fetch a backend's specification through the Admin API:

```bash
curl -i http://localhost:30001/api/admin/oas/echo \
  -b 'session=<admin session cookie>'
```

- Requires the superuser or an admin user with the `oas-viewer` permission. Otherwise the response is `401` or `403`.
- Returns `200` with `Content-Type: application/yaml` and the file contents as they are on disk.
- Returns `404` if the backend has no mapping, or if the `oas` section is not configured.

See [Admin API](./admin-api.md) for logging in and managing permissions. The `GET /api/admin/flow` endpoint (`flow-viewer` permission) also lists which backends have validation enabled, under the `oas-validator` step.

## Things to know

- **Set `validateBody` explicitly** whenever you include `options`.
- **Check your specification before you deploy.** Kerberos stops at startup if the file cannot be read or parsed. Lint your specification in CI as well.
- **Match backend names exactly.** `oas.mappings[].backend` must name a configured backend. Use `${ref:...}` references or check `GET /api/admin/flow` after deploying.
- **One specification per backend.** Add one mapping for each backend.
- **Restart after changing a specification.** The gateway reads the file at startup.
- **List every operation.** Any path or method you leave out of the specification returns `404` at the gateway, including `HEAD` and plain `OPTIONS` requests.

## Related docs

- [Routing](./routing.md): URL format, path stripping and forwarding.
- [Authentication](./authentication.md): access control, which validation does not provide.
- [Admin API](./admin-api.md): permissions, including `oas-viewer` and `flow-viewer`.
- [Debugging](./admin-debugging.md): tracing a request through the pipeline.
- [Configuration](./configuration.md): file format and `${ref:...}` references.
- Example specifications: `test/oas/echo.yaml`, `test/oas/echo-protected.yaml`.
