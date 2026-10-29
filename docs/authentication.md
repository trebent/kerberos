# Authentication and authorisation

Kerberos can protect the backends behind the gateway so that only logged-in users reach them, and optionally only users in specific groups. You enable it with the `auth` section of the configuration file: Kerberos then checks every gateway request for a mapped backend before forwarding it, and tells the backend who the caller is through `X-Krb-*` request headers.

## How it works

Kerberos currently offers one authentication method, **basic**:

1. Users belong to an **organisation** and can be members of **groups** within it. Organisations, users and groups are managed through the basic-auth API. See [Organisations](./organizations.md).
2. A user logs in by sending a username and password to the basic-auth API on the **admin port** (`POST /api/auth/basic/organisations/{orgID}/login`). Kerberos answers with a `session` cookie (plus `refresh` and `csrf` cookies). See [Sessions](./sessions.md) for cookie lifetimes, refresh and logout.
3. The client calls a backend through the **gateway port** (`/gw/backend/<name>/...`) and sends the `session` cookie along.
4. Kerberos looks up the session, applies the backend's authorisation rules, adds identity headers and forwards the request. Requests that fail the checks are answered by Kerberos with `401` or `403` and never reach the backend.

Backends that have no entry in `auth.scheme.mappings` are not protected at all.

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

Leave the whole `auth` section out to disable authentication. If you include it, include both `methods` and `scheme`: the schema requires them together, and an `auth` section without `methods` makes Kerberos fail at startup.

| Key | Type | Default | Description |
|---|---|---|---|
| `methods` | object | - | Authentication methods to enable. Required together with `scheme`. |
| `methods.basic` | object | - | Enables the basic method and its API on the admin port. `{}` is enough to enable it with defaults. |
| `methods.basic.api.cookies` | object | `sameSite: Strict`, no domain | `domain` and `sameSite` (`Strict`, `Lax`, `None`) for the session, refresh and csrf cookies. See [Sessions](./sessions.md). |
| `methods.basic.api.origins` | object | none (no CORS headers) | Which browser origins may call the basic-auth API: one of `allowedOrigins`, `allowAll` or `denyAll`. See [CORS](./cors.md). |
| `scheme` | object | - | Which backends are protected and how. Required together with `methods`. |
| `scheme.mappings` | array | - | At least one mapping. See below. |
| `order` | integer | `0` when omitted | Position of the auth check among the optional pipeline steps (auth and [request validation](./request-validation.md)). Lower runs first. The schema says the minimum is `1`. |

The order matters when you also use request validation: if auth runs first, an unauthenticated request gets `401`; if validation runs first, a malformed request gets `400` before its session is checked. Set `order` explicitly on both sections so the order is not left to chance.

### `scheme.mappings[]`

| Key | Type | Required | Description |
|---|---|---|---|
| `backend` | string | yes | Name of a backend in `gateway.router.backends`. You can use `${ref:gateway.router.backends[1].name}` to avoid repeating it (see [Configuration](./configuration.md)). |
| `method` | string | yes | Authentication method. Only `basic` is supported. |
| `exempt` | array of strings | no | Path patterns that skip authentication and authorisation completely. |
| `authorization` | object | no | Group requirements. Without it, any logged-in user is allowed. |
| `authorization.groups` | array of strings | no (min. 1 item if set) | Group names required for every path of the backend. The user must be in **any one** of them. |
| `authorization.paths` | object | no | Map of path pattern to array of group names (min. 1 item). For matching paths these groups are required **instead of** `authorization.groups`. |

Use one mapping per backend. If a backend is listed more than once, the exemptions of the first mapping and the authorisation of the last mapping are used, which is rarely what you want.

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
- The whole path must match. `*` matches any characters except `/`, so it never spans more than one path segment. There is no `**`; to cover a subtree, list each depth you need (`/public/*`, `/public/*/*`) or structure the backend's paths so one level is enough.
- To cover both a path and its children, list both, as in `["/unprotected", "/unprotected/*"]`.
- The query string is not part of the path and is ignored.
- Patterns are not checked when the configuration is loaded. A malformed pattern (for example an unclosed `[`) in `exempt` makes every request to that backend fail with `500`; in `authorization.paths` it makes requests fail with `403`.

## Authorisation semantics

- **Any of the listed groups.** A user needs to be a member of at least one listed group, not all of them.
- **Groups are matched by name.** Kerberos looks up the groups the user belongs to in their own organisation and compares group names. Two organisations that both have a group called `staff` both satisfy `"groups": ["staff"]`.
- **Path rules replace global groups.** If a path matches an `authorization.paths` pattern, only that pattern's groups count; `authorization.groups` is ignored for that request. In the example above, a user who is only in `staff` gets `403` on `/long/hello`.
- **Avoid overlapping path patterns.** If more than one pattern in `authorization.paths` matches a path, which one is used is not defined and can differ between requests. Make the patterns mutually exclusive.
- **No rules means any logged-in user.** Without `authorization`, or when neither `authorization.groups` is set nor a path rule matches, every authenticated user is allowed, whatever their organisation.
- Group membership is read on every request, so adding a user to or removing a user from a group takes effect on their next request without logging in again.

## Identity headers sent to backends

On requests that pass the session check, Kerberos adds these headers before forwarding:

| Header | Value | When it is set |
|---|---|---|
| `X-Krb-User` | Numeric user ID, e.g. `7` | Every request that passes the session check. |
| `X-Krb-Org` | Numeric organisation ID, e.g. `1` | Every request that passes the session check. |
| `X-Krb-Groups` | Group name; one header line per group the user belongs to | Only when a group check ran for the request (the mapping has `authorization` and either a path rule matched or `authorization.groups` is set). Not added if the user has no groups. |

None of these headers are added for unmapped backends or exempt paths. If your backend needs the user's groups on every protected path, configure `authorization.groups` for the backend.

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

Each getter returns `false` when the header was absent. The middleware does no verification of its own: it trusts whatever headers arrive, so it is only safe when the backend can be reached through Kerberos alone (see below).

## Security considerations

Read this section before relying on Kerberos to protect a backend.

- **Backends must only be reachable through Kerberos.** Kerberos is the only thing enforcing authentication. If clients can reach a backend directly, they can bypass every check and send any `X-Krb-*` headers they like. Put backends on a private network, behind a firewall, or use [mutual TLS](./tls.md) between Kerberos and the backend.
- **Incoming `X-Krb-*` headers are not removed.** Kerberos does not strip identity headers sent by the client:
  - On unmapped backends and on exempt paths, a client can send its own `X-Krb-User`, `X-Krb-Org` and `X-Krb-Groups`, and they reach the backend as-is. Do not treat these headers as trustworthy on routes that are not protected.
  - On protected paths, `X-Krb-User` and `X-Krb-Org` are overwritten with the real values, but client-sent `X-Krb-Groups` values are kept: Kerberos appends the real groups when a group check runs and adds nothing when it does not. Access decisions in Kerberos itself use the stored group membership, but a backend that reads `X-Krb-Groups` may see extra, client-chosen values. If your backend makes its own decisions based on groups, enforce those groups in Kerberos configuration instead, or make sure clients cannot add the header (for example with a proxy in front of Kerberos that removes `X-Krb-*` request headers).
- **A session from any organisation authenticates.** Kerberos does not tie a backend to an organisation. Any user of any organisation can log in and pass the session check. Since groups are matched by name and organisation administrators can create groups freely, group rules alone do not stop another organisation from creating a group with the same name. If a backend belongs to one organisation, check `X-Krb-Org` in the backend as well.
- **Always configure `methods.basic` when a mapping uses `"method": "basic"`.** The configuration is accepted without it, but every request to the mapped backend then fails instead of being authenticated.
- **The session cookie is sent to the backend.** Because headers are forwarded unchanged, backends receive the user's `session` cookie. Do not log it, and treat backends as trusted parties.
- **Use HTTPS.** The session cookies are marked `Secure`, so browsers only send them over HTTPS (except to `localhost`). See [TLS](./tls.md) and [Sessions](./sessions.md).
- Exempt patterns are matched before anything else; double-check that a broad pattern such as `/*` does not expose more than you intended.

## Walk-through with curl

This assumes the example configuration above, Kerberos on `localhost` with the gateway on port 30000 and the admin API on port 30001, and an organisation with ID `1` containing a user `alice` who is a member of group `staff`. See [Organisations](./organizations.md) for creating the organisation, user and group. The session cookie is `Secure`; curl sends it over plain HTTP only to `localhost`, so use HTTPS for any other host.

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

The session expires after a short time; refresh it or log in again as described in [Sessions](./sessions.md).

## Related docs

- [Organisations](./organizations.md): organisations, users, groups and the basic-auth API
- [Sessions](./sessions.md): cookies, refresh, logout and CSRF
- [CORS](./cors.md): allowing browser origins for the basic-auth API and for backends
- [Routing](./routing.md): backends, path handling and the request pipeline
- [Request validation](./request-validation.md): OpenAPI validation and its `order`
- [Admin API](./admin-api.md): platform administration, including creating organisations
- [TLS](./tls.md): HTTPS for the gateway and admin API, and mTLS to backends
