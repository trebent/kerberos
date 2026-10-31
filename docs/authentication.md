# Authentication and authorisation

Kerberos can protect the backends behind the gateway so that only logged-in users reach them, and optionally only users in specific groups. You enable it with the `auth` section of the configuration file: Kerberos then checks every gateway request for a mapped backend before forwarding it, and tells the backend who the caller is through `X-Krb-*` request headers.

## How it works

Kerberos logs users in with the **basic** method, using a username and password:

1. Users belong to an **organisation** and can be members of **groups** within it. Organisations, users and groups are managed through the basic-auth API. See [Organisations](./organizations.md).
2. A user logs in by sending a username and password to the basic-auth API on the **admin port** (`POST /api/auth/basic/organisations/{orgID}/login`). Kerberos answers with a `session` cookie (plus `refresh` and `csrf` cookies). See [Sessions](./sessions.md) for cookie lifetimes, refresh and logout.
3. The client calls a backend through the **gateway port** (`/gw/backend/<name>/...`) and sends the `session` cookie along.
4. Kerberos looks up the session, applies the backend's authorisation rules, adds identity headers and forwards the request. Requests that fail the checks are answered by Kerberos with `401` or `403` and never reach the backend.

Backends that have no entry in `auth.scheme.mappings` are open to everyone.

The basic-auth API is only served when `auth.methods.basic` is configured. Its OpenAPI contract is [`openapi/auth_basic.yaml`](../openapi/auth_basic.yaml), and `auth_basic.yaml` must be present in the OAS directory at startup.

## Configuration

Example, based on the configuration used by the integration tests:

```json
{
  "gateway": {
    "router": {
      "backends": [
        { "name": "echo", "host": "echo", "port": 15000 },
        { "name": "protected-echo", "host": "protected-echo", "port": 15000 }
      ]
    }
  },
  "auth": {
    "methods": {
      "basic": {
        "api": {
          "cookies": { "sameSite": "Strict" },
          "origins": { "allowedOrigins": ["https://app.example.com"] }
        }
      }
    },
    "scheme": {
      "mappings": [
        {
          "backend": "protected-echo",
          "method": "basic",
          "exempt": ["/unprotected", "/unprotected/*"],
          "authorization": {
            "groups": ["staff"],
            "paths": {
              "/long/hello": ["pleb"]
            }
          }
        }
      ]
    },
    "order": 1
  }
}
```

With this configuration:

- `echo` is not mapped, so anyone can call it.
- `/gw/backend/protected-echo/unprotected` and anything directly below `/unprotected/` are open to everyone.
- `/gw/backend/protected-echo/long/hello` needs a logged-in user in the group `pleb`.
- Every other path on `protected-echo` needs a logged-in user in the group `staff`.

The authoritative schema is [`internal/config/schemas/auth_schema.json`](../internal/config/schemas/auth_schema.json).

### `auth`

Leave the whole `auth` section out to turn authentication off. If you include it, include both `methods` and `scheme`.

| Key | Type | Default | Description |
|---|---|---|---|
| `methods` | object | - | Authentication methods to enable. Required together with `scheme`. |
| `methods.basic` | object | - | Enables the basic method and its API on the admin port. `{}` is enough to enable it with defaults. |
| `methods.basic.api.cookies` | object | see [Sessions: cookie configuration](./sessions.md#cookie-configuration) | Attributes of the session, refresh and CSRF cookies set by basic-auth logins. |
| `methods.basic.api.origins` | object | see [CORS](./cors.md#the-origins-object) | Which browser origins may call the basic-auth API. |
| `scheme` | object | - | Which backends are protected and how. Required together with `methods`. |
| `scheme.mappings` | array | - | At least one mapping. See below. |
| `order` | integer | see [Routing: pipeline order](./routing.md#pipeline-order) | Position of the auth check relative to [request validation](./request-validation.md). Set it explicitly. |

### `scheme.mappings[]`

| Key | Type | Required | Description |
|---|---|---|---|
| `backend` | string | yes | Name of a backend in `gateway.router.backends`. You can use `${ref:gateway.router.backends[1].name}` to avoid repeating it (see [Configuration](./configuration.md)). |
| `method` | string | yes | Authentication method: `basic`. |
| `exempt` | array of strings | no | Path patterns that skip authentication and authorisation completely. |
| `authorization` | object | no | Group requirements. Without it, any logged-in user is allowed. |
| `authorization.groups` | array of strings | no (min. 1 item if set) | Group names required for every path of the backend. The user must be in **any one** of them. |
| `authorization.paths` | object | no | Map of path pattern to array of group names (min. 1 item). For matching paths these groups are required **instead of** `authorization.groups`. |

Use one mapping per backend.

## What happens on each request

For each request to `/gw/backend/<name>/...`, Kerberos does the following after routing (see [Routing](./routing.md) for the full pipeline):

1. **Backend not mapped**: the request is forwarded unchanged.
2. **Path matches an `exempt` pattern**: the request is forwarded unchanged. No session is needed and no identity headers are added.
3. **Session check**: the request must carry a non-empty `session` cookie that belongs to an existing, unexpired basic-auth session. Otherwise Kerberos responds `401` with `{"errors":["Unauthorized"]}`. On success, `X-Krb-User` and `X-Krb-Org` are set on the forwarded request.
4. **Authorisation**:
   - If the mapping has no `authorization`, the request is allowed.
   - If the path matches a pattern in `authorization.paths`, the user must be in one of that pattern's groups.
   - Otherwise, if `authorization.groups` is set, the user must be in one of those groups.
   - Otherwise (no global groups and no matching path rule) the request is allowed.

   When a group check runs and the user is in none of the required groups, Kerberos responds `403` with `{"errors":["Forbidden"]}`.
5. The request is passed on to the next pipeline step and then forwarded to the backend.

An admin API session (see [Admin API](./admin-api.md)) is not a basic-auth session and does not grant access to protected backends.

## Path patterns

`exempt` entries and `authorization.paths` keys use Go [`path.Match`](https://pkg.go.dev/path#Match) glob syntax:

| Pattern | Matches | Does not match |
|---|---|---|
| `/health` | `/health` | `/health/`, `/health/live` |
| `/public/*` | `/public/a`, `/public/index.html` | `/public`, `/public/a/b` |
| `/v?/items` | `/v1/items`, `/v2/items` | `/v10/items` |
| `/files/[a-f]*` | `/files/abc` | `/files/xyz` |

Things to know:

- Patterns are matched against the path **relative to the backend**, after Kerberos has removed `/gw/backend/<name>`. A request to `/gw/backend/protected-echo/unprotected/x` is matched as `/unprotected/x`. Do not include the `/gw/backend/<name>` prefix in patterns.
- The whole path must match. `*` matches any characters except `/`, so it covers one path segment. To cover deeper paths, list each depth you need (`/public/*`, `/public/*/*`).
- To cover both a path and its children, list both, as in `["/unprotected", "/unprotected/*"]`.
- Patterns match the path only, not the query string.
- Test new patterns before you deploy them.

## Authorisation semantics

- **Any of the listed groups.** A user needs to be a member of at least one listed group, not all of them.
- **Groups are matched by name.** Kerberos looks up the groups the user belongs to in their own organisation and compares them with the group names in the rule.
- **Path rules replace global groups.** If a path matches an `authorization.paths` pattern, only that pattern's groups count; `authorization.groups` is ignored for that request. In the example above, a user who is only in `staff` gets `403` on `/long/hello`.
- **Avoid overlapping path patterns.** Make the patterns in `authorization.paths` mutually exclusive, so that each path matches at most one rule.
- **No rules means any logged-in user.** Without `authorization`, or when neither `authorization.groups` is set nor a path rule matches, every authenticated user is allowed.
- Group membership is read on every request, so adding a user to or removing a user from a group takes effect on their next request without logging in again.

## Identity headers sent to backends

On requests that pass the session check, Kerberos adds these headers before forwarding:

| Header | Value | When it is set |
|---|---|---|
| `X-Krb-User` | Numeric user ID, e.g. `7` | Every request that passes the session check. |
| `X-Krb-Org` | Numeric organisation ID, e.g. `1` | Every request that passes the session check. |
| `X-Krb-Groups` | Group name; one header line per group the user belongs to | When a group check ran for the request (the mapping has `authorization` and either a path rule matched or `authorization.groups` is set) and the user is in at least one group. |

These headers are added on protected paths, not on unmapped backends or exempt paths. If your backend needs the user's groups on every protected path, configure `authorization.groups` for the backend.

All other request headers, including the `Cookie` header with the session cookie, are forwarded to the backend as well.

### Reading the headers in a Go backend

Go backends can use the `github.com/trebent/kerberos/pkg/auth` package. `ContextMiddleware` copies the three headers into the request context, and `UserFromContext`, `OrgFromContext` and `GroupsFromContext` read them back as strings:

```go
package main

import (
	"fmt"
	"net/http"

	krbauth "github.com/trebent/kerberos/pkg/auth"
)

func main() {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /hello", func(w http.ResponseWriter, r *http.Request) {
		user, ok := krbauth.UserFromContext(r.Context())
		if !ok {
			http.Error(w, "unauthenticated", http.StatusUnauthorized)
			return
		}
		org, _ := krbauth.OrgFromContext(r.Context())
		groups, _ := krbauth.GroupsFromContext(r.Context())

		fmt.Fprintf(w, "user %s in organisation %s, groups %v\n", user, org, groups)
	})

	http.ListenAndServe(":15000", krbauth.ContextMiddleware()(mux))
}
```

Each getter returns `false` when the header is missing. Use it in backends that are reached through Kerberos (see below).

## Good practice

Follow these points when Kerberos protects a backend.

- **Backends must only be reachable through Kerberos.** Kerberos is what enforces authentication, so every request to a protected backend must pass through it. Put backends on a private network, behind a firewall, or use [mutual TLS](./tls.md) between Kerberos and the backend.
- **Rely on identity headers only on protected routes.** `X-Krb-*` headers describe the caller only on requests that Kerberos authenticated.
- **Separate tenants in the backend.** If a backend belongs to one organisation, check `X-Krb-Org` in the backend as well as using group rules.
- **Configure `methods.basic` whenever a mapping uses `"method": "basic"`.**
- **Use HTTPS.** The session cookies are marked `Secure`, so browsers only send them over HTTPS (except to `localhost`). See [TLS](./tls.md) and [Sessions](./sessions.md).
- **Keep exempt patterns narrow.** Exempt patterns are checked first, so make sure a broad pattern such as `/*` opens only what you intend.

## Walkthrough with curl

This assumes the example configuration above, Kerberos on `localhost` on the [default ports](./README.md#conventions), and an organisation with ID `1` containing a user `alice` who is a member of group `staff`. See [Organisations](./organizations.md) for creating the organisation, user and group. The session cookie is `Secure`; curl sends it over plain HTTP only to `localhost`, so use HTTPS for any other host.

1. Call the protected backend without a session:

   ```bash
   curl -i http://localhost:30000/gw/backend/protected-echo/
   # HTTP/1.1 401 Unauthorized
   # {"errors":["Unauthorized"]}
   ```

2. Exempt paths work without a session:

   ```bash
   curl -i http://localhost:30000/gw/backend/protected-echo/unprotected
   # HTTP/1.1 200 OK
   ```

3. Log in on the admin port and store the cookies in a cookie jar:

   ```bash
   curl -i -c cookies.txt \
     -H 'Content-Type: application/json' \
     -d '{"username":"alice","password":"correct-horse-battery"}' \
     http://localhost:30001/api/auth/basic/organisations/1/login
   # HTTP/1.1 204 No Content
   # Set-Cookie: session=...; Path=/; Max-Age=900; HttpOnly; Secure; SameSite=Strict
   # Set-Cookie: refresh=...
   # Set-Cookie: csrf=...
   ```

4. Call the protected backend on the gateway port with the session cookie:

   ```bash
   curl -i -b cookies.txt http://localhost:30000/gw/backend/protected-echo/
   # HTTP/1.1 200 OK
   ```

   The backend receives `X-Krb-User`, `X-Krb-Org: 1` and `X-Krb-Groups: staff`.

5. Call the path that requires `pleb`, which `alice` is not a member of:

   ```bash
   curl -i -b cookies.txt http://localhost:30000/gw/backend/protected-echo/long/hello
   # HTTP/1.1 403 Forbidden
   # {"errors":["Forbidden"]}
   ```

To keep using the backend after the session expires, refresh it or log in again as described in [Sessions](./sessions.md).

## Related docs

- [Organisations](./organizations.md): organisations, users, groups and the basic-auth API
- [Sessions](./sessions.md): cookies, refresh, logout and CSRF
- [CORS](./cors.md): allowing browser origins for the basic-auth API and for backends
- [Routing](./routing.md): backends, path handling and the request pipeline
- [Request validation](./request-validation.md): OpenAPI validation and its `order`
- [Admin API](./admin-api.md): platform administration, including creating organisations
- [TLS](./tls.md): HTTPS for the gateway and admin API, and mTLS to backends
