# Routing

Kerberos sends each request to one of your services based on its path. You list your services as named backends, and clients reach a backend by putting its name in the request path. Kerberos then runs the request through its checks and forwards it to that backend. This page covers how to define backends, how URLs map to them, what reaches the backend, timeouts and the errors Kerberos returns.

## Defining backends

Backends are listed under `gateway.router.backends`. You need at least one.

```json
{
  "gateway": {
    "router": {
      "backends": [
        {
          "name": "echo",
          "host": "localhost",
          "port": 15000
        },
        {
          "name": "orders-api",
          "host": "orders.internal",
          "port": 8443,
          "timeout": 15000,
          "origins": {
            "allowedOrigins": ["https://app.example.com"]
          },
          "tls": {
            "rootCAFile": "/etc/krb/certs/internal-ca.pem"
          }
        }
      ]
    }
  }
}
```

With this configuration, `GET http://localhost:30000/gw/backend/orders-api/orders/42` is forwarded to `https://orders.internal:8443/orders/42`.

### Backend keys

| Key | Type | Default | Description |
| --- | --- | --- | --- |
| `name` | string, 1–100 characters | required | The name that clients put in the URL. Use lowercase letters, digits, `-` and `_`. See [Backend names](#backend-names). |
| `host` | string, 1–256 characters | required | Hostname or IP address of the backend. |
| `port` | integer, 1–65535 | required | Port of the backend. |
| `timeout` | integer, milliseconds, at least 1 | `5000` | The longest a forwarded request may take. See [Timeouts](#timeouts). |
| `origins` | object | none | Which browser origins may call this backend. When you leave it out, Kerberos does no origin checks and sends no CORS headers for this backend. See [CORS](./cors.md). |
| `tls` | object | none | Use HTTPS to reach the backend, with optional mutual TLS. Without it, Kerberos uses plain HTTP. See [Backend TLS](#backend-tls). |

Kerberos rejects unknown keys at startup. The authoritative schema is [`internal/config/schemas/router_schema.json`](../internal/config/schemas/router_schema.json). For config file loading and `${env:...}` / `${ref:...}` references, see [Configuration](./configuration.md).

### Backend names

Backend names use **lowercase letters `a-z`, digits `0-9`, `-` and `_`**, for example `orders-api` or `orders_v2`.

Names are compared exactly. Give each backend a unique name.

## URL format

All proxied traffic goes to the gateway port (`PORT`, see [Deployment](./deployment.md#environment-variables)) under this prefix:

```
/gw/backend/<backend-name>/<backend-path>
```

- `<backend-name>` is the `name` of a configured backend.
- `<backend-path>` is the path the backend receives, and it may be empty.
- The `/` after the backend name is required. `/gw/backend/echo` returns `400`, and `/gw/backend/echo/` reaches the backend root.

### Path stripping

Kerberos removes `/gw/backend/<backend-name>` from the path before forwarding. The backend sees only the remainder.

| Request to Kerberos | Path the `echo` backend receives |
| --- | --- |
| `/gw/backend/echo/` | `/` |
| `/gw/backend/echo/hi` | `/hi` |
| `/gw/backend/echo/api/v1/users/7` | `/api/v1/users/7` |
| `/gw/backend/echo` | not forwarded: `400` |

Kerberos does not add a prefix of its own. If your backend serves its API under `/api`, clients must include it: `/gw/backend/echo/api/...`.

The [request validation](./request-validation.md) and [authentication](./authentication.md) settings, such as OpenAPI paths, exempt paths and authorisation path patterns, all match against the stripped, backend-relative path.

### Other paths on the gateway port

| Path | Result |
| --- | --- |
| `/gw/health` | `204 No Content` for any method. See [Health endpoint](#health-endpoint). |
| Any other path under `/gw/`, such as `/gw/foo` | `400` with a JSON error |
| Anything outside `/gw/` | `404` with a plain-text body `404 page not found` |

Paths that contain `//`, `.` or `..` segments are cleaned up first: the gateway answers with a `307` redirect to the clean path, for example `/gw/backend/echo//x` redirects to `/gw/backend/echo/x`.

The admin and login APIs run on a separate port (`ADMIN_PORT`). See [Admin API](./admin-api.md).

## Request lifecycle

Every request under `/gw/backend/` passes through these stages in order. Any stage can answer the request itself and stop it there.

1. **Observability.** Kerberos starts a trace (or continues the caller's trace if the request has a W3C `traceparent` header), and starts counting metrics and logging the request. It also reads the backend name from the path, and a badly formed path gets `400` here. See [Observability](./observability.md).
2. **Router.** Kerberos looks up the backend by name, and an unknown name gets `404`. If the request has an `Origin` header and the backend has an `origins` block, Kerberos checks the origin. A disallowed origin gets `403`. An allowed origin gets CORS headers, and a browser's preflight `OPTIONS` request is answered with `204` without reaching the backend. See [CORS](./cors.md). The router then strips the path prefix.
3. **Access control and request validation**, when configured. [Authentication](./authentication.md) (the `auth` section) and [request validation](./request-validation.md) (the `oas` section) run in the order set by each section's `order` value. See [Pipeline order](#pipeline-order). They apply only to backends that are mapped to them.
4. **Forwarding.** Kerberos sends the request to the backend and relays the response.

If you are debugging a request, an [admin debug session](./admin-debugging.md) records how it moves through each of these stages.

### Pipeline order

`auth.order` and `oas.order` place authentication and request validation in stage 3. Both use the same key, defined by [`ordered_schema.json`](../internal/config/schemas/ordered_schema.json):

| Key | Type | Default | Description |
| --- | --- | --- | --- |
| `order` | integer, at least 1 | `0` when omitted | Position among the optional steps. Lower values run first. |

```text
router -> (auth and oas, sorted by order) -> forward to backend
```

- **Set `order` on both sections, with different values**, so the order is always clear.
- **Run authentication first**, for example `"order": 1` on `auth` and `"order": 100` on `oas`. Callers who are not logged in then get `401` before anything else, and learn nothing about which paths your API has.

## What is forwarded

Kerberos builds the upstream URL as `http://<host>:<port><backend-path>`, or `https://` when the backend has a `tls` block, and forwards:

- **Method.** The method is forwarded unchanged.
- **Path.** Kerberos forwards the stripped backend path. Percent-encoded characters are decoded and encoded again, so an encoded slash (`%2F`) reaches the backend as `/`.
- **Headers.** Client headers are forwarded as-is, including `Authorization`, `Cookie`, `Origin` and any `X-Forwarded-*` headers the client sent itself.
- **`Host`.** `Host` is set to the backend's `host:port`, not the host the client used.
- **Body.** The body is streamed to the backend with `Transfer-Encoding: chunked`.
- **Trace context.** When observability is enabled, Kerberos sets the W3C `traceparent` and `baggage` headers so the backend continues Kerberos' trace. When observability is disabled, trace headers sent by the client pass through unchanged.
- **Identity headers.** When the backend is protected by [authentication](./authentication.md), Kerberos adds `X-Krb-Org`, `X-Krb-User` and, when group authorisation applies, `X-Krb-Groups` for an authenticated caller.

## Response relay

Kerberos returns the backend's status code, headers and body to the client unchanged and streams the body.

When CORS is configured for the backend and the request had an allowed `Origin`, Kerberos adds its CORS headers to the response before the backend's headers. When Kerberos handles CORS for a backend, turn CORS handling off in the backend itself. See [CORS](./cors.md).

## Timeouts

Each backend has a `timeout` in milliseconds (see [Backend keys](#backend-keys) for the default). It limits the **whole** upstream exchange: connecting, sending the request, waiting for the response and reading the response body. When it runs out, the client gets `500` with `{"errors":["failed to forward request"]}`.

The gateway server's own read and write timeouts (`READ_TIMEOUT_SECONDS` and `WRITE_TIMEOUT_SECONDS`) also apply to every request. Keep the write timeout above the backend `timeout`. For slow backends, raise both. See [Deployment: timeouts](./deployment.md#timeouts).

## Backend TLS

To reach a backend over HTTPS, optionally with mutual TLS, add a `tls` block to the backend. Its keys are documented in [TLS: backend TLS and mutual TLS](./tls.md#backend-tls-and-mutual-tls).

## Health endpoint

`/gw/health` on the gateway port returns `204 No Content` with no body, for any HTTP method. It shows that the gateway server is accepting requests. Use it for container health checks and load-balancer probes:

```bash
curl -i http://localhost:30000/gw/health
# HTTP/1.1 204 No Content
```

## Errors

Errors produced by Kerberos itself use `Content-Type: application/json` and this body:

```json
{"errors":["<message>"]}
```

Responses from your backend, including its error responses, are relayed unchanged and do not use this format.

| Status | Message | Cause |
| --- | --- | --- |
| `400` | `failed to extract backend name from request path` | The path does not match `/gw/backend/<name>/...`: the slash after the name is missing, the name has characters outside `a-z`, `0-9`, `-` and `_`, or the path is not a backend path (for example `/gw/foo`). |
| `404` | `no backend found` | No backend with that name is configured. |
| `403` | `origin denied` | The request's `Origin` is not allowed by the backend's `origins` settings. See [CORS](./cors.md). |
| `401` / `403` | `Unauthorized` / `Forbidden` | Rejected by [authentication](./authentication.md): no valid session, or the caller is not in a required group. |
| `400` / `404` | validation message | Rejected by [request validation](./request-validation.md): the request does not match the backend's OpenAPI spec (`400`), or the path or method is not in the spec (`404`). |
| `500` | `failed to forward request` | Kerberos could not complete the exchange with the backend: connection refused, DNS failure, TLS error, or the backend `timeout` ran out. |

The Kerberos logs record the exact cause of a `500`.

## Example

Start the [echo test backend](./echo.md), which replies with the method, path, headers and body it received, then run Kerberos from source:

```bash
make echo/run   # echo backend on port 15000
make run        # Kerberos with test/config/local.json
```

`test/config/local.json` protects the `echo` backend with authentication and request validation. To try plain forwarding first, use a minimal config with only `gateway.router.backends` (see [Getting started](./getting-started.md)). Then:

```bash
curl -s http://localhost:30000/gw/backend/echo/a/b -H "X-Custom: hi"
```

```json
{
  "method": "GET",
  "url": "/a/b",
  "headers": {
    "Accept": ["*/*"],
    "Accept-Encoding": ["gzip"],
    "User-Agent": ["curl/8.5.0"],
    "X-Custom": ["hi"]
  }
}
```

The prefix has been stripped and the custom header has arrived.

## Related docs

- [Configuration](./configuration.md): the config file and all sections
- [CORS](./cors.md): per-backend `origins`
- [Authentication](./authentication.md): protecting backends and identity headers
- [Request validation](./request-validation.md): OpenAPI validation per backend
- [Observability](./observability.md): metrics, traces and logs per backend
- [TLS](./tls.md): gateway HTTPS and backend TLS/mTLS
- [Deployment](./deployment.md): ports, environment variables and health checks
- [Gateway API reference](https://trebent.github.io/kerberos/api/gateway): the rendered [`openapi/gateway.yaml`](../openapi/gateway.yaml) contract
