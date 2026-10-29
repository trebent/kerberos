# Origins and CORS

Kerberos can decide which browser origins (web applications on other sites) may call the gateway and its APIs. You configure this with an `origins` object per backend and per API; Kerberos then answers CORS preflight requests itself, adds the CORS response headers browsers need, and rejects requests from origins you have not allowed.

If you never call Kerberos from JavaScript running on another origin, you can leave `origins` unset and skip this page.

## How it works

Browsers attach an `Origin` header (for example `Origin: https://app.example.com`) to cross-origin requests, and to most same-origin requests that are not `GET` or `HEAD`. Kerberos uses only that header:

- A request **without** an `Origin` header is never subject to origin checks. Server-to-server calls, `curl`, and most same-origin `GET` requests pass straight through.
- A request **with** an `Origin` header is checked against the `origins` setting that applies to it (see [Where origins can be set](#where-origins-can-be-set)). Depending on the mode, the request is rejected with `403`, or CORS headers are added to the response and the request continues.
- An `OPTIONS` request with an allowed `Origin` is treated as a CORS preflight and answered by Kerberos directly with `204 No Content`.

## The `origins` object

The same object is used everywhere. It has three mutually exclusive modes:

| Key | Type | Default | Description |
| --- | --- | --- | --- |
| `allowedOrigins` | array of strings | `[]` | Origins that are allowed. Each entry is compared to the request's `Origin` header by exact, case-sensitive string match. Any other origin gets `403`. |
| `allowAll` | boolean | `false` | Allow every origin. The request's `Origin` is echoed back in `Access-Control-Allow-Origin`. |
| `denyAll` | boolean | `false` | Reject every request that carries an `Origin` header with `403`. |

Rules enforced by the configuration schema ([`internal/config/schemas/origins_schema.json`](../internal/config/schemas/origins_schema.json)):

- At most one mode may be active. `allowAll: true` together with `denyAll: true` or a non-empty `allowedOrigins` fails validation at startup, and so does `denyAll: true` with a non-empty `allowedOrigins`.
- Unknown keys are rejected.
- Setting a mode to its "off" value is allowed, for example `"allowAll": false`.

An `origins` object with no active mode (`{}`, `"allowedOrigins": []`, or all flags `false`) behaves exactly like leaving `origins` out: no origin handling at all.

### Writing `allowedOrigins` entries

An origin is `scheme://host[:port]`, exactly as the browser sends it:

| Entry | Matches `Origin: https://app.example.com`? |
| --- | --- |
| `https://app.example.com` | Yes |
| `https://app.example.com/` | No (trailing slash) |
| `https://app.example.com:443` | No (browsers leave out default ports) |
| `http://app.example.com` | No (different scheme) |
| `https://*.example.com` | No (wildcards are not supported) |
| `https://App.Example.com` | No (comparison is case-sensitive) |

List every origin explicitly, including any non-default port, for example `http://localhost:4200` for a local frontend dev server.

## Where origins can be set

| Location | Applies to | Port |
| --- | --- | --- |
| `gateway.router.backends[].origins` | Requests to that backend, `/gw/backend/<name>/...` | Gateway (30000) |
| `admin.api.origins` | The admin API, `/api/admin/...` | Admin (30001) |
| `auth.methods.basic.api.origins` | The basic-auth API, `/api/auth/basic/...` | Admin (30001) |

Each location is independent. Allowing an origin on the admin API does not allow it on the basic-auth API or on any backend, and each backend has its own setting.

The [admin-connector](./admin-connector.md) has its own `origins` setting with the same format and behaviour as the admin API.

## Configuration example

```json
{
  "gateway": {
    "router": {
      "backends": [
        {
          "name": "shop",
          "host": "shop",
          "port": 8080,
          "origins": {
            "allowedOrigins": ["https://shop.example.com", "http://localhost:4200"]
          }
        },
        {
          "name": "public-catalog",
          "host": "catalog",
          "port": 8080,
          "origins": { "allowAll": true }
        },
        {
          "name": "internal",
          "host": "internal",
          "port": 8080,
          "origins": { "denyAll": true }
        }
      ]
    }
  },
  "admin": {
    "api": {
      "origins": { "allowedOrigins": ["https://console.example.com"] }
    }
  },
  "auth": {
    "methods": {
      "basic": {
        "api": {
          "origins": { "allowedOrigins": ["https://shop.example.com"] }
        }
      }
    },
    "scheme": {
      "mappings": [{ "backend": "shop", "method": "basic" }]
    }
  }
}
```

In this example:

- The `shop` frontend at `https://shop.example.com` can log in through the basic-auth API and then call the `shop` backend. A local dev server on `http://localhost:4200` can call the `shop` backend but not the basic-auth API.
- Any site can call `public-catalog`.
- No browser page on any origin can call `internal`; non-browser clients still can.
- Only `https://console.example.com` can call the admin API from a browser.

See [Configuration](./configuration.md) for the rest of the file format.

## Behaviour

### Summary

| `origins` mode | Request has no `Origin` | `OPTIONS` with `Origin` | Other method with `Origin` |
| --- | --- | --- | --- |
| Not set / empty | Passes through | Gateway: forwarded like any other request. APIs: `204`, no CORS headers | Passes through, no CORS headers |
| `denyAll` | Passes through | `403` | `403` |
| `allowAll` | Passes through | `204` with CORS headers | CORS headers added, request continues |
| `allowedOrigins`, origin listed | Passes through | `204` with CORS headers | CORS headers added, request continues |
| `allowedOrigins`, origin not listed | Passes through | `403` | `403` |

### When `origins` is not set

Kerberos does no CORS handling: the `Origin` header is not checked and no `Access-Control-*` headers are added.

- **Gateway:** the request, including any `OPTIONS` preflight and its `Origin` header, goes through the normal pipeline ([authentication](./authentication.md), [request validation](./request-validation.md)) and on to the backend unchanged. If the backend implements CORS itself, its headers reach the browser. Browsers do not send cookies on preflight requests, so if the backend is protected by authentication the preflight will usually be rejected with `401` and the browser will block the real request.
- **Admin and basic-auth APIs:** an `OPTIONS` request is answered with `204` but without CORS headers, so browsers treat the preflight as failed. Cross-origin calls from a browser therefore do not work until you configure `origins`.

### Rejected origins

A rejected request gets `403` with a JSON error body and no CORS headers. Because there are no CORS headers, a browser shows it as a CORS failure and the page's JavaScript cannot read the status or body.

- Gateway: `{"errors":["origin denied"]}`
- Admin and basic-auth APIs: `{"errors":["Forbidden"]}`

`denyAll` rejects every request with an `Origin` header, including requests made from a page on the same origin. Browsers send `Origin` on same-origin `POST`, `PUT`, `PATCH` and `DELETE` requests, so do not use `denyAll` on a backend that serves a browser application which calls it. Use `allowedOrigins` with that application's own origin instead.

### Allowed origins: response headers

When the origin is allowed, Kerberos sets these headers on the response:

| Header | Value |
| --- | --- |
| `Access-Control-Allow-Origin` | The request's `Origin`, echoed back (never `*`) |
| `Access-Control-Allow-Methods` | `GET, POST, PUT, PATCH, DELETE, OPTIONS` |
| `Access-Control-Allow-Headers` | `Content-Type, X-Krb-Csrf-Token` |
| `Access-Control-Allow-Credentials` | `true` |

These values are fixed and cannot be configured. In particular:

- **Only `Content-Type` and `X-Krb-Csrf-Token` may be sent as non-simple request headers.** A browser will block a cross-origin request that sets `Authorization`, `X-Request-Id`, or any other custom header. Kerberos authentication uses cookies, not the `Authorization` header (see [Sessions](./sessions.md)).
- **`HEAD` is not listed** in the allowed methods. Browsers do not need a preflight for simple `HEAD` requests, so this matters only if the request also carries a non-simple header.
- **Credentials are always allowed**, so browsers send and accept cookies on cross-origin requests when the page uses `credentials: "include"` (or `withCredentials`).
- **No `Access-Control-Max-Age`** is sent, so browsers cache preflight results only for their built-in default (a few seconds), and send a preflight before most non-simple requests.
- **No `Access-Control-Expose-Headers`** is sent, so page scripts can only read the CORS-safelisted response headers (such as `Content-Type`), not custom headers returned by your backend.
- **No `Vary: Origin`** is added. If a shared cache sits between browsers and Kerberos, make sure it does not serve a response cached for one origin to another.

The headers are added before the rest of the pipeline runs, so error responses produced later by Kerberos (for example `401` from authentication or `400` from request validation) also carry them, and the page can read those errors.

Kerberos does not compare the preflight's `Access-Control-Request-Method` or `Access-Control-Request-Headers` with what it allows. It always returns the fixed list above and leaves the decision to the browser.

### Preflight requests

Any `OPTIONS` request with an allowed `Origin` is answered by Kerberos with `204 No Content` and the headers above. It never reaches authentication, request validation, or the backend. This also means a backend behind an origin-enabled route never receives `OPTIONS` requests from browsers.

An `OPTIONS` request without an `Origin` header is not a preflight. On the gateway it is handled like any other request and forwarded to the backend.

### Where the check happens

On the gateway the origin check runs right after Kerberos has found the backend from the URL, and before [authentication](./authentication.md), [request validation](./request-validation.md), and forwarding. The order is:

1. The path is parsed; a malformed path (`400`) or unknown backend (`404`) is rejected first, without CORS headers.
2. The `Origin` is checked against that backend's `origins`; rejected (`403`), or CORS headers are set and preflights answered (`204`).
3. Authentication and request validation run, if configured.
4. The request is forwarded to the backend.

On the admin and basic-auth APIs the origin check also runs first: before request validation against the API's OpenAPI contract, before the CSRF check, and before the session is checked.

See [Routing](./routing.md) for the full gateway pipeline.

### Backends that also send CORS headers

When `origins` is configured for a backend, Kerberos sets the CORS headers itself and then adds the backend's response headers on top. If the backend also sends `Access-Control-Allow-Origin` or other CORS headers, the browser receives the header twice and will reject the response. Disable CORS handling in the backend when Kerberos handles it, or leave the backend's `origins` unset and let the backend handle CORS on its own.

## Examples

The examples assume Kerberos runs locally with the gateway on port 30000 and the admin API on port 30001, and the `shop` backend from the configuration example above.

### Preflight to an allowed origin

```bash
curl -i -X OPTIONS http://localhost:30000/gw/backend/shop/orders \
  -H 'Origin: https://shop.example.com' \
  -H 'Access-Control-Request-Method: POST' \
  -H 'Access-Control-Request-Headers: content-type'
```

```http
HTTP/1.1 204 No Content
Access-Control-Allow-Credentials: true
Access-Control-Allow-Headers: Content-Type, X-Krb-Csrf-Token
Access-Control-Allow-Methods: GET, POST, PUT, PATCH, DELETE, OPTIONS
Access-Control-Allow-Origin: https://shop.example.com
```

(Other headers such as `Date` are omitted here and below.)

### Preflight from an origin that is not allowed

```bash
curl -i -X OPTIONS http://localhost:30000/gw/backend/shop/orders \
  -H 'Origin: https://evil.example.net' \
  -H 'Access-Control-Request-Method: POST'
```

```http
HTTP/1.1 403 Forbidden
Content-Type: application/json

{"errors":["origin denied"]}
```

### Actual cross-origin request

```bash
curl -i http://localhost:30000/gw/backend/shop/orders \
  -H 'Origin: https://shop.example.com'
```

The response is whatever the backend returns (or an authentication or validation error from Kerberos), plus the four `Access-Control-*` headers.

### Preflight to the basic-auth API

```bash
curl -i -X OPTIONS http://localhost:30001/api/auth/basic/organisations/1/login \
  -H 'Origin: https://shop.example.com' \
  -H 'Access-Control-Request-Method: POST' \
  -H 'Access-Control-Request-Headers: content-type'
```

This returns `204` with the same four headers. With `auth.methods.basic.api.origins` unset it returns `204` without any `Access-Control-*` headers, and a browser would block the login call.

### Calling Kerberos from a browser application

```js
// Log in through the basic-auth API (admin port). Login is exempt from CSRF.
await fetch("https://krb.example.com:30001/api/auth/basic/organisations/1/login", {
  method: "POST",
  credentials: "include",
  headers: { "Content-Type": "application/json" },
  body: JSON.stringify({ username: "alice", password: "..." }),
});

// Call a protected backend through the gateway. The session cookie is sent
// because of credentials: "include".
const res = await fetch("https://krb.example.com:30000/gw/backend/shop/orders", {
  credentials: "include",
});
```

Whether the browser actually sends the session cookie across sites also depends on the cookie's `SameSite` and `Domain` settings. See [Sessions](./sessions.md).

## CORS and CSRF

CORS controls which origins can read responses and make non-simple requests. It is separate from the CSRF protection on the admin and basic-auth APIs:

- On the admin and basic-auth APIs, a state-changing request (anything other than `GET`, `HEAD`, `OPTIONS`, `TRACE`) that carries an `Origin` header must also send the `X-Krb-Csrf-Token` header with the value of the `csrf` cookie, or it is rejected with `403`. That is why `X-Krb-Csrf-Token` is in `Access-Control-Allow-Headers`. Login endpoints are exempt. The origin check runs first, so a request from a disallowed origin gets `403` before the CSRF check.
- The gateway does no CSRF checking. For backends protected by [authentication](./authentication.md), the defences against cross-site requests are the `SameSite` attribute of the session cookie (default `Strict`) and the backend's `origins` setting.

See [Sessions](./sessions.md) for how the CSRF token and cookies work.

## Things to know

- **`allowAll` sends credentials to any site.** Because `Access-Control-Allow-Credentials: true` is always set, `allowAll` lets any website make cookie-carrying requests and read the responses, as far as the browser's `SameSite` rules allow. Use `allowAll` only for public, unauthenticated backends. Prefer `allowedOrigins` for anything behind authentication, and for the admin and basic-auth APIs.
- **The check relies on the `Origin` header.** Non-browser clients can leave it out or set any value they like, so origin settings protect browser users, not the backend. They do not replace authentication.
- **`Origin: null`** (sent by sandboxed iframes and some redirects and `file://` pages) is treated like any other value: echoed back with `allowAll`, allowed only if `"null"` is listed in `allowedOrigins`, and rejected with `denyAll`.
- **The settings are fixed for the whole location.** You cannot set different origins for different paths of one backend, or change the allowed headers or methods.
- **Admin and basic-auth APIs share a port but not a setting.** If a browser application uses both APIs, list its origin in both `admin.api.origins` and `auth.methods.basic.api.origins`.
- **Denied and preflight requests are still counted** in the gateway's request metrics, under the backend label `unknown`. See [Observability](./observability.md).

## Related docs

- [Sessions](./sessions.md): cookies, `SameSite`, and CSRF tokens
- [Authentication](./authentication.md): protecting backends with login
- [Routing](./routing.md): backend configuration and the gateway pipeline
- [Admin API](./admin-api.md) and [Organisations](./organizations.md): the APIs on the admin port
- [Configuration](./configuration.md): the configuration file
