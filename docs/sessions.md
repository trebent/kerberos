# Sessions and cookies

Kerberos has two login APIs: the basic-auth API, where organisation users sign in, and the Admin API, where admin users and the superuser sign in. Both keep the login state in cookies backed by a session stored in the database. This page covers how those sessions start, refresh and end, which cookies Kerberos sets, how CSRF protection works, and how to configure cookies so that browsers and scripts keep their sessions.

## Overview

Both APIs run on the admin port (30001 in the examples). The lifecycle is the same for both:

1. **Login.** Send your credentials. Kerberos creates a session in the database and returns three cookies: `session`, `refresh` and `csrf`.
2. **Use the session.** Send the `session` cookie on later requests. Organisation users also send it to the gateway (port 30000) to reach protected backends (see [Authentication](./authentication.md)).
3. **Refresh.** Before you lose access, call the refresh endpoint with the `refresh` cookie. Kerberos deletes the old session, creates a new one and replaces all three cookies.
4. **Logout.** Kerberos deletes the session and tells the client to delete the `session` and `refresh` cookies.

### Endpoints

| Step | Basic-auth API (organisation users) | Admin API: admin users | Admin API: superuser |
|---|---|---|---|
| Login | `POST /api/auth/basic/organisations/{orgID}/login` with `{"username", "password"}` | `POST /api/admin/login` with `{"username", "password"}` | `POST /api/admin/superuser/login` with `{"clientId", "clientSecret"}` |
| Who am I | `GET /api/auth/basic/organisations/{orgID}/users/me` | `GET /api/admin/me` | `GET /api/admin/me` |
| Refresh | `POST /api/auth/basic/organisations/{orgID}/refresh` | `POST /api/admin/refresh` | `POST /api/admin/superuser/refresh` |
| Logout | `POST /api/auth/basic/organisations/{orgID}/logout` | `POST /api/admin/logout` | `POST /api/admin/superuser/logout` |

A successful login, refresh or logout returns `204 No Content`. Wrong credentials return `401`, and so does a refresh that is no longer allowed. For the rest of each API, see [Organisations](./organizations.md) and the [Admin API](./admin-api.md).

## Cookies

Every successful login and refresh sets these cookies:

| Cookie | Lifetime (`Max-Age`) | HttpOnly | Secure | Path | Purpose |
|---|---|---|---|---|---|
| `session` | 15 minutes | yes | yes | `/` | Identifies the session. Sent on every API call and, for organisation users, on gateway requests. |
| `refresh` | 1 hour | yes | yes | Only the refresh endpoint (see below) | Used only to get a new session. |
| `csrf` | 1 hour | no | yes | `/` | Random value for CSRF double-submit. Not HttpOnly so browser scripts can read it. |

Each API scopes the `refresh` cookie to its own refresh path, so browsers only send it to that one endpoint:

| Login type | `refresh` cookie path |
|---|---|
| Organisation user | `/api/auth/basic/organisations/{orgID}/refresh` (with that organisation's ID) |
| Admin user | `/api/admin/refresh` |
| Superuser | `/api/admin/superuser/refresh` |

All three cookies also get the `SameSite` and `Domain` attributes from your configuration (see [Cookie configuration](#cookie-configuration)).

On logout, Kerberos expires the `session` and `refresh` cookies. The `csrf` cookie is left in place and stays until its own lifetime runs out. That does no harm, because it does not give access to anything.

### The Secure flag needs HTTPS

All three cookies always carry the `Secure` flag, and you cannot turn it off. Browsers and most HTTP clients keep and send `Secure` cookies only over HTTPS. The exception is `localhost` and loopback addresses, which browsers, curl and Go's `net/http/cookiejar` treat as secure even over plain HTTP. If you reach Kerberos by any other hostname, serve the admin port over HTTPS (`admin.api.tls`), or put it behind a TLS-terminating proxy. Otherwise logins will appear to work but no session will stick. See [TLS](./tls.md).

## Server-side session lifetime

Each login or refresh creates a session row in the database that expires **15 minutes** after it was created. The cookie lifetime matches that. After 15 minutes the session no longer authenticates anything, even if a client still sends the cookie, and the APIs and gateway answer `401`. Activity does not extend a session. The only way to extend it is to refresh it.

Sessions live in the configured database (see [Persistence](./persistence.md)), so they survive a restart and all Kerberos instances sharing that database see them.

## Refreshing a session

A refresh needs only the `refresh` cookie. You do not need a valid `session` cookie. On success, Kerberos:

- deletes the old session, so the old `session` cookie stops working right away,
- creates a new 15-minute session, and
- sets a new `session`, `refresh` and `csrf` cookie. The old refresh value cannot be used again.

When a refresh is allowed depends on the API:

| API | Refresh allowed |
|---|---|
| Basic-auth API | Only while the current session is still valid, so within 15 minutes of the last login or refresh. Once the session has expired, the user has to log in again. |
| Admin API (admin users and superuser) | While the session is valid, and for up to **45 minutes after it expires**. That is up to 60 minutes after the last login or refresh, which matches the 1-hour `refresh` cookie. |

Further rules:

- A basic-auth refresh token only works on the refresh endpoint of the organisation it was issued for.
- A superuser refresh token only works on `/api/admin/superuser/refresh`. Using it on the admin-user refresh path fails because of the cookie path. Using an admin user's token on the superuser path returns `403`.

Good practice for clients: refresh a little before the 15-minute mark, for example every 10 to 12 minutes, or refresh when a call returns `401` (this works for admin sessions only, because basic-auth sessions cannot be refreshed once expired).

## Logging out

| Logout | Effect |
|---|---|
| Organisation user (`.../organisations/{orgID}/logout`) | Deletes only the current session. Other sessions of the same user, for example on other devices, stay valid. |
| Admin user (`/api/admin/logout`) | Deletes only the current session. |
| Superuser (`/api/admin/superuser/logout`) | Deletes **all** superuser sessions, including those on other devices and in other scripts. Calling it with an admin-user session returns `403`. |

## CSRF protection

Both APIs use the double-submit pattern. Kerberos checks a request for CSRF only when **both** of these are true:

- it carries an `Origin` header, which browsers add to cross-origin requests and to most same-origin requests that are not `GET`, and
- its method is not one of `GET`, `HEAD`, `OPTIONS` or `TRACE`.

The request must then include an `X-Krb-Csrf-Token` header whose value equals the `csrf` cookie. If the header or the cookie is missing, or their values differ, Kerberos returns `403 {"errors":["Forbidden"]}`.

The login endpoints (`/api/auth/basic/organisations/{orgID}/login`, `/api/admin/login` and `/api/admin/superuser/login`) are exempt, because no `csrf` cookie exists before login. Refresh and logout are **not** exempt.

Requests without an `Origin` header are never CSRF-checked. That covers most scripts, curl and server-to-server clients, unless they set `Origin` themselves. CSRF protection applies only to the basic-auth and Admin APIs. Requests forwarded by the gateway to your backends are not CSRF-checked by Kerberos.

### Browser example

The `csrf` cookie is readable from JavaScript. Copy its value into the header on every unsafe request:

```js
function csrfToken() {
  const match = document.cookie.match(/(?:^|;\s*)csrf=([^;]*)/);
  return match ? decodeURIComponent(match[1]) : "";
}

// Log in (no CSRF header needed)
await fetch("https://admin.example.com:30001/api/auth/basic/organisations/1/login", {
  method: "POST",
  credentials: "include",
  headers: { "Content-Type": "application/json" },
  body: JSON.stringify({ username: "alice", password: "correct-horse-1" }),
});

// Any later POST/PUT/PATCH/DELETE, including refresh and logout
await fetch("https://admin.example.com:30001/api/auth/basic/organisations/1/refresh", {
  method: "POST",
  credentials: "include",
  headers: { "X-Krb-Csrf-Token": csrfToken() },
});
```

`document.cookie` only shows the `csrf` cookie if the page's host can see it. That means the page must be on the same host as the API, or on a host covered by the configured cookie `domain`. If the page runs on another origin, you also need CORS for the API. `X-Krb-Csrf-Token` is always among the allowed headers. See [CORS](./cors.md).

### curl example

curl sends no `Origin` header by default, so CSRF checks do not apply. If you add `Origin` yourself, for example to test CORS, take the token from the cookie jar:

```sh
# Log in and store cookies in a jar
curl -s -c jar.txt -b jar.txt \
  -H 'Content-Type: application/json' \
  -d '{"username":"alice","password":"correct-horse-1"}' \
  http://localhost:30001/api/auth/basic/organisations/1/login

# Read the csrf value from the jar (7th column of the Netscape cookie file)
CSRF=$(awk '$6 == "csrf" { print $7 }' jar.txt)

# Unsafe request with an Origin header: the CSRF header is required
curl -s -c jar.txt -b jar.txt -X POST \
  -H 'Origin: http://localhost:8080' \
  -H "X-Krb-Csrf-Token: $CSRF" \
  http://localhost:30001/api/auth/basic/organisations/1/logout
```

## Cookie configuration

You configure cookie attributes separately for each API:

- `admin.api.cookies` applies to Admin API logins (admin users and superuser).
- `auth.methods.basic.api.cookies` applies to basic-auth logins.

```json
{
  "admin": {
    "api": {
      "cookies": {
        "domain": "admin.example.com",
        "sameSite": "Strict"
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
          }
        }
      }
    }
  }
}
```

| Key | Type | Default | Description |
|---|---|---|---|
| `domain` | string | none (host-only cookie) | Becomes the cookie's `Domain` attribute. With no domain, the browser returns the cookie only to the exact host that set it. With a domain, it also sends it to subdomains of that domain. |
| `sameSite` | `Strict` \| `Lax` \| `None` | `Strict` | Becomes the cookie's `SameSite` attribute. Use `Lax` or `None` only when a page on another site needs to send the cookies. |

Things to know about these settings:

- Kerberos checks `sameSite` against the schema at startup. Any value other than `Strict`, `Lax` or `None` (case-sensitive) stops Kerberos from starting with a schema validation error. Values are never replaced silently. The schema is `internal/config/schemas/cookies_schema.json`.
- If you configure `auth.methods.basic.api.cookies` but leave out `sameSite`, for example to set only `domain`, Kerberos sends the basic-auth cookies **with no `SameSite` attribute**, and the browser's default applies (usually `Lax`). Leaving out the whole `cookies` block gives `Strict`. Always set `sameSite` explicitly when you set `domain`. For `admin.api.cookies`, a missing `sameSite` stays `Strict`.
- Browsers accept `SameSite=None` only on `Secure` cookies. Kerberos cookies are always `Secure`, so `None` works over HTTPS.

Example: organisation users log in through the admin port at `auth.example.com` and call the gateway at `api.example.com`. Cookies ignore ports, so on a single hostname the gateway port receives the `session` cookie automatically. With two hostnames, set `auth.methods.basic.api.cookies.domain` to `example.com` so the browser sends `session` to both.

## Admin and basic-auth sessions share the `session` cookie

Admin sessions and basic-auth sessions use the same cookie name, `session`, with `Path=/`. Browsers key cookies by name, domain and path, but **not by port**. In one browser, a login on one API therefore overwrites the `session` cookie of the other when both are served from the same host (or both use the same cookie `domain`):

- Logging in to the Admin API (for example, the admin UI) replaces an organisation user's `session` cookie. Gateway requests and basic-auth API calls then use the admin session. Gateway routes protected by basic auth answer `401`.
- Logging in as an organisation user replaces the admin `session` cookie, and Admin API calls answer `401`.
- While the `session` cookie holds a superuser session, or an admin session with the `basic-auth-org-admin` or `basic-auth-org-viewer` permission, the basic-auth refresh endpoint answers `401` even if a valid basic-auth `refresh` cookie is sent.

The `refresh` cookies do not collide, because their paths differ. The `csrf` cookie is shared too, but any matching value passes the CSRF check, so that alone causes no failures.

Ways to avoid this:

- **Separate hostnames.** Reach the Admin API and the basic-auth flow through different hostnames, for example `admin.example.com` and `auth.example.com` pointing at the same admin port. Leave `domain` unset, or set each API's `cookies.domain` to its own host, so that neither cookie is visible to the other host. Do not set a shared parent domain (such as `example.com`) on both APIs.
- **Separate browser profiles** (or a private window) for admin work and for testing as an organisation user.
- **Separate cookie jars** for scripts. Use one jar per identity, as in the examples below.

An admin session (superuser, or an admin user with the `basic-auth-org-admin` or `basic-auth-org-viewer` permission) is itself accepted by the basic-auth API for managing organisations (see [Organisations](./organizations.md)). It is not accepted by the gateway for basic-auth-protected routes.

## Using cookie jars

### curl

Use `-c` to save cookies and `-b` to send them, with one jar file per identity:

```sh
# Superuser
curl -s -c admin.jar -b admin.jar -H 'Content-Type: application/json' \
  -d '{"clientId":"admin","clientSecret":"secret"}' \
  http://localhost:30001/api/admin/superuser/login
curl -s -b admin.jar http://localhost:30001/api/admin/me

# Refresh: curl sends the refresh cookie only to the path it was set for
curl -s -c admin.jar -b admin.jar -X POST http://localhost:30001/api/admin/superuser/refresh

# Organisation user, in its own jar
curl -s -c alice.jar -b alice.jar -H 'Content-Type: application/json' \
  -d '{"username":"alice","password":"correct-horse-1"}' \
  http://localhost:30001/api/auth/basic/organisations/1/login
curl -s -b alice.jar http://localhost:30000/gw/backend/echo/hello
```

Always pass `-c` on refresh, so the jar picks up the new cookies. The old ones stop working as soon as the refresh succeeds. Over plain HTTP, curl stores and sends these `Secure` cookies only for `localhost` and loopback addresses. For other hosts, use HTTPS.

### Go clients

The generated clients `github.com/trebent/kerberos/pkg/auth/client` (basic-auth API) and `github.com/trebent/kerberos/pkg/admin/client` (Admin API) do not manage cookies themselves. Give them an `http.Client` that has a cookie jar:

```go
import (
	"context"
	"net/http"
	"net/http/cookiejar"

	authclient "github.com/trebent/kerberos/pkg/auth/client"
)

jar, _ := cookiejar.New(nil)
httpClient := &http.Client{Jar: jar}

c, err := authclient.NewClientWithResponses(
	"http://localhost:30001",
	authclient.WithHTTPClient(httpClient),
)
if err != nil {
	return err
}

ctx := context.Background()
_, err = c.LoginWithResponse(ctx, 1, authclient.LoginJSONRequestBody{
	Username: "alice",
	Password: "correct-horse-1",
})
// Later calls, including RefreshWithResponse and LogoutWithResponse,
// send and update the cookies through the jar.
```

The same `httpClient` can call the gateway on port 30000, because cookie jars ignore ports. Use a separate jar for each identity, and never share one jar between an admin login and an organisation-user login, for the reason explained under [shared cookie name](#admin-and-basic-auth-sessions-share-the-session-cookie). If your client sets an `Origin` header, also add `X-Krb-Csrf-Token` with a request editor (`WithRequestEditorFn`) that copies the `csrf` value from the jar.

## Things to know

- Sessions last 15 minutes, and activity does not extend them. Clients must refresh.
- Basic-auth sessions cannot be refreshed after they expire. Admin sessions can be refreshed for up to 45 minutes after they expire.
- Superuser logout ends every superuser session. Other logouts end only the current session.
- Cookies are always `Secure`, so use HTTPS for anything other than `localhost`.
- Admin and basic-auth logins overwrite each other's `session` cookie on the same host.
- Kerberos does not purge expired sessions or refresh tokens from the database. They stop working but remain stored.

## Related docs

- [Authentication](./authentication.md): protecting gateway backends with basic-auth sessions.
- [Organisations](./organizations.md): the basic-auth API, organisations, users and groups.
- [Admin API](./admin-api.md): superuser, admin users and permissions.
- [CORS](./cors.md): allowing browser pages on other origins to call the APIs.
- [TLS](./tls.md): serving the admin port over HTTPS.
- [Configuration](./configuration.md): the full configuration file.
- OpenAPI contracts: [`../openapi/auth_basic.yaml`](../openapi/auth_basic.yaml) and [`../openapi/admin.yaml`](../openapi/admin.yaml).
