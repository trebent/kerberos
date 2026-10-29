# Routing

Kerberos is a path-based reverse proxy. You list your services as named backends, and clients reach a backend by putting its name in the request path. Kerberos then runs the request through its checks and forwards it to that backend. This page covers how to define backends, how URLs map to them, what reaches the backend, timeouts and the errors Kerberos returns.

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
| `name` | string, 1–100 characters | required | The name that clients put in the URL. Only lowercase letters, digits, `-` and `_` can be routed to. See [Backend names](#backend-names). |
| `host` | string, 1–256 characters | required | Hostname or IP address of the backend. |
| `port` | integer, 1–65535 | required | Port of the backend. |
| `timeout` | integer, milliseconds, at least 1 | `5000` | The longest a forwarded request may take. See [Timeouts](#timeouts). |
| `origins` | object | none | Which browser origins may call this backend. When you leave it out, Kerberos does no origin checks and sends no CORS headers for this backend. See [CORS](./cors.md). |
| `tls` | object | none | Use HTTPS to reach the backend, with optional mutual TLS. Without it, Kerberos uses plain HTTP. See [Backend TLS](#backend-tls). |

Kerberos rejects unknown keys at startup. The authoritative schema is [`internal/config/schemas/router_schema.json`](../internal/config/schemas/router_schema.json). For config file loading and `${env:...}` / `${ref:...}` references, see [Configuration](./configuration.md).

### Backend names

Kerberos reads the backend name from the URL with a strict pattern: **only lowercase letters `a-z`, digits `0-9`, `-` and `_`**. The configuration schema accepts other names, such as `Orders` or `orders.v2`, but requests to them can never be routed and always get `400`. Use lowercase names such as `orders-api` or `orders_v2`.

Names are compared exactly. Give each backend a unique name. Kerberos does not reject duplicates, and if two backends share a name, the settings that are used get mixed between the two entries.

## URL format

All proxied traffic goes to the gateway port (`PORT`, default `30000`) under this prefix:

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
| `/gw/backend/echo/a/b?x=1` | `/a/b` (the query string is dropped, see [Limitations](#limitations)) |
| `/gw/backend/echo` | not forwarded: `400` |

Kerberos never adds a prefix of its own. If your backend serves its API under `/api`, clients must include it: `/gw/backend/echo/api/...`.

The [request validation](./request-validation.md) and [authentication](./authentication.md) settings, such as OpenAPI paths, exempt paths and authorisation path patterns, all match against the stripped, backend-relative path.

### Other paths on the gateway port

| Path | Result |
| --- | --- |
| `/gw/health` | `204 No Content` for any method. See [Health endpoint](#health-endpoint). |
| Any other path under `/gw/`, such as `/gw/foo` | `400` with a JSON error |
| Anything outside `/gw/` | `404` with a plain-text body `404 page not found` |

Paths that contain `//`, `.` or `..` segments are not forwarded as sent. Instead the gateway answers with a `307` redirect to the cleaned path, for example `/gw/backend/echo//x` redirects to `/gw/backend/echo/x`.

The admin and login APIs run on a separate port (`ADMIN_PORT`, default `30001`). See [Admin API](./admin-api.md).

## Request lifecycle

Every request under `/gw/backend/` passes through these stages in order. Any stage can answer the request itself and stop it there.

1. **Observability.** Kerberos starts a trace span (it continues an incoming W3C `traceparent` if one is present), and starts counting metrics and request logging. It also reads the backend name from the path, and a malformed path is rejected here with `400`. See [Observability](./observability.md).
2. **Router.** Kerberos looks up the backend by name, and an unknown name gets `404`. If the request has an `Origin` header and the backend has an `origins` block, Kerberos checks the origin. A disallowed origin gets `403`. An allowed origin gets CORS headers, and a preflight `OPTIONS` request is answered with `204` without reaching the backend. See [CORS](./cors.md). The router then strips the path prefix.
3. **Access control and request validation**, when configured. [Authentication](./authentication.md) (the `auth` section) and [request validation](./request-validation.md) (the `oas` section) run in the order set by each section's `order` value, lowest first. They apply only to backends that are mapped to them. Use different values, for example `auth` `1` and `oas` `100`. Running authentication first means that unauthenticated callers get `401` and never learn anything from validation errors.
4. **Forwarding.** Kerberos sends the request to the backend and relays the response.

If you are debugging a request, an [admin debug session](./admin-debugging.md) records how it moves through each of these stages.

## What is forwarded

Kerberos builds the upstream URL as `http://<host>:<port><backend-path>`, or `https://` when the backend has a `tls` block, and forwards:

- **Method.** The method is forwarded unchanged.
- **Path.** Kerberos forwards the stripped backend path. Percent-encoded characters are decoded and then re-encoded, so an encoded slash (`%2F`) reaches the backend as a real `/`.
- **Headers.** Client headers are forwarded as-is, including `Authorization`, `Cookie`, `Origin` and any `X-Forwarded-*` headers the client sent itself.
- **`Host`.** `Host` is set to the backend's `host:port`, not the host the client used.
- **Body.** The body is streamed to the backend. When a body is present, it is sent with `Transfer-Encoding: chunked` instead of the client's `Content-Length`.
- **Trace context.** When observability is enabled, Kerberos sets the W3C `traceparent` and `baggage` headers so the backend continues Kerberos' trace. When observability is disabled, trace headers sent by the client pass through unchanged.
- **Identity headers.** When the backend is protected by [authentication](./authentication.md), Kerberos adds `X-Krb-Org`, `X-Krb-User` and, when group authorisation applies, `X-Krb-Groups` for an authenticated caller.

Kerberos does **not** forward the query string, and it does **not** add `X-Forwarded-For`, `X-Forwarded-Host`, `X-Forwarded-Proto`, `Forwarded` or `Via` headers. See [Limitations](#limitations).

## Response relay

Kerberos returns the backend's status code, headers and body to the client unchanged and streams the body. Kerberos does not rewrite redirects (`Location` headers), cookies or response bodies.

When CORS is configured for the backend and the request had an allowed `Origin`, Kerberos adds its CORS headers to the response before the backend's headers. If the backend also sets CORS headers, the client receives both sets. Turn off CORS handling in the backend when Kerberos handles it. See [CORS](./cors.md).

## Timeouts

Each backend has a `timeout` in milliseconds (default `5000`). It limits the **whole** upstream exchange: connecting, sending the request, waiting for the response and reading the response body. When it runs out, the client gets `500` with `{"errors":["failed to forward request"]}`.

The gateway server's own timeouts also apply to every request:

| Environment variable | Default | Effect |
| --- | --- | --- |
| `READ_TIMEOUT_SECONDS` | `5` | Maximum time to read the client's request, including the body. |
| `WRITE_TIMEOUT_SECONDS` | `5` | Maximum time from the end of reading the request headers to the end of writing the response. |

`WRITE_TIMEOUT_SECONDS` puts a ceiling on the backend `timeout`. If a backend takes longer than the write timeout, Kerberos still waits for it, up to the backend `timeout`. The client's connection is then closed with no response at all, not a JSON error. For slow backends, raise both settings, and keep `WRITE_TIMEOUT_SECONDS` above the largest backend `timeout`:

```json
{ "name": "reports", "host": "reports.internal", "port": 8080, "timeout": 30000 }
```

```bash
WRITE_TIMEOUT_SECONDS=35 ./kerberos --config config.json
```

These server timeouts also apply to the admin port. See [Deployment](./deployment.md) for all environment variables.

## Backend TLS

To reach a backend over HTTPS, add a `tls` block. `rootCAFile` is required whenever `tls` is present. Set it to an empty string to trust the system CA pool. For mutual TLS, add `clientCertFile` and `clientKeyFile` together.

```json
"tls": {
  "rootCAFile": "/etc/krb/certs/internal-ca.pem",
  "clientCertFile": "/etc/krb/certs/krb-client.pem",
  "clientKeyFile": "/etc/krb/certs/krb-client-key.pem"
}
```

Certificate files are loaded at startup, so a missing or invalid file stops Kerberos from starting. See [TLS](./tls.md) for all options, including `insecureSkipVerify`, and for serving the gateway itself over HTTPS.

## Health endpoint

`/gw/health` on the gateway port returns `204 No Content` with no body, for any HTTP method. It does not check backends or the database. It only shows that the gateway server is accepting requests. Use it for container health checks and load-balancer probes:

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
| `400` | `failed to extract backend name from request path` | The path does not match `/gw/backend/<name>/...`: the trailing slash is missing, the name contains uppercase letters or other unsupported characters, or the path is not a backend path at all (for example `/gw/foo`). |
| `404` | `no backend found` | No backend with that name is configured. |
| `403` | `origin denied` | The request's `Origin` is not allowed by the backend's `origins` settings. See [CORS](./cors.md). |
| `401` / `403` | `Unauthorized` / `Forbidden` | Rejected by [authentication](./authentication.md): no valid session, or the caller is not in a required group. |
| `400` / `404` | validation message | Rejected by [request validation](./request-validation.md): the request does not match the backend's OpenAPI spec (`400`), or the path or method is not in the spec (`404`). |
| `500` | `failed to forward request` | Kerberos could not complete the exchange with the backend: connection refused, DNS failure, TLS error, or the backend `timeout` ran out. |

Kerberos does not use `502 Bad Gateway` or `504 Gateway Timeout`. Every upstream failure is reported as `500`. The Kerberos logs record the actual cause.

## Example

Start the bundled echo backend, which replies with the method, path, headers and body it received, then run Kerberos from source:

```bash
make echo/run   # echo backend on port 15000
make run        # Kerberos with test/config/local.json
```

`test/config/local.json` protects the `echo` backend with authentication and request validation. To try plain forwarding first, use a minimal config with only `gateway.router.backends` (see [Getting started](./getting-started.md)). Then:

```bash
curl -s "http://localhost:30000/gw/backend/echo/a/b?x=1" -H "X-Custom: hi"
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

The prefix has been stripped, the custom header has arrived, and `?x=1` was dropped.

## Limitations

- **The query string is dropped.** Only the path is forwarded, so `?x=1` never reaches the backend. Backends that depend on query parameters, such as filtering or pagination, need to take them in the path, headers or body instead.
- **No `X-Forwarded-*` headers.** The backend sees Kerberos as the client and gets `Host` set to its own address. It cannot tell the original client IP, host or scheme unless the client sends those headers itself. Client-supplied values are forwarded unchecked, so do not trust them.
- **Client headers are passed through.** Kerberos does not remove headers that the client sent. Backends should rely only on the `X-Krb-*` identity headers of requests that Kerberos actually authenticated, and should be reachable only through Kerberos. See [Authentication](./authentication.md).
- **Every upstream failure is `500`.** Timeouts, refused connections and TLS errors all look the same to the client.
- **Backend names must be lowercase.** Names with uppercase letters, dots or other characters pass config validation but cannot be routed to.
- **Request bodies are sent chunked.** Backends that require a `Content-Length` header on requests will reject forwarded bodies.
- **Server write timeout caps backend timeouts.** See [Timeouts](#timeouts).
- **One path prefix per backend.** There is no host-based routing, no path rewriting beyond stripping the prefix, and no load balancing across multiple hosts for a backend. Put a load balancer or service address in `host` if you need that.

## Related docs

- [Configuration](./configuration.md): the config file and all sections
- [CORS](./cors.md): per-backend `origins`
- [Authentication](./authentication.md): protecting backends and identity headers
- [Request validation](./request-validation.md): OpenAPI validation per backend
- [Observability](./observability.md): metrics, traces and logs per backend
- [TLS](./tls.md): gateway HTTPS and backend TLS/mTLS
- [Deployment](./deployment.md): ports, environment variables and health checks
