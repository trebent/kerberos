# Admin API

The Admin API is how you administer a running Kerberos instance: log in as the superuser, create admin users and groups, grant them permissions, inspect the active request flow and the OpenAPI specifications used for validation, and run debug sessions. It is served on the admin port (`ADMIN_PORT`, default `30001`) under `/api/admin`, separate from the gateway port that serves your backends.

The contract for every endpoint is [`../openapi/admin.yaml`](../openapi/admin.yaml). This page explains the concepts and the behaviour around it.

## Overview

- **Always enabled.** The Admin API has no on/off switch. Every Kerberos instance serves it on the admin port. Kerberos reads `admin.yaml` from the OpenAPI directory (`OAS_DIRECTORY`) at startup and fails to start if that file is missing (see [Deployment](./deployment.md)).
- **Same port as the basic-auth API.** If basic authentication is configured, its API (`/api/auth/basic/...`) is served on the same admin port. The admin port's TLS settings cover both APIs.
- **Cookie-based sessions.** You log in with a JSON request and receive `session`, `refresh` and `csrf` cookies. Every later request sends those cookies. See [Sessions](./sessions.md).
- **Two kinds of identity:**
  - The **superuser**: a single built-in account that has every permission.
  - **Admin users**: accounts you create. They get permissions only through the **admin groups** they belong to.

Admin users, admin groups and the superuser are separate from the users and groups of basic-auth organisations described in [Organisations](./organizations.md). An admin user cannot log in to a backend through the gateway, and an organisation user cannot log in to the Admin API.

## Configuration

The `admin` section of the configuration file is optional. With no `admin` section, the Admin API runs with the defaults below.

```json
{
  "admin": {
    "superUser": {
      "clientId": "admin",
      "clientSecret": "${env:KRB_SUPERUSER_SECRET}"
    },
    "api": {
      "cookies": {
        "domain": "admin.example.com",
        "sameSite": "Strict"
      },
      "origins": {
        "allowedOrigins": ["https://admin-ui.example.com"]
      },
      "tls": {
        "serverCertFile": "/etc/kerberos/tls/admin.crt",
        "serverKeyFile": "/etc/kerberos/tls/admin.key"
      }
    }
  }
}
```

| Key | Type | Default | Description |
| --- | --- | --- | --- |
| `admin.superUser.clientId` | string | `admin` | Superuser login name. Used only on the first start (see [The superuser](#the-superuser)). |
| `admin.superUser.clientSecret` | string | `secret` | Initial superuser password. Used only on the first start. |
| `admin.api.cookies.domain` | string | none (host-only cookies) | `Domain` attribute for the session, refresh and CSRF cookies. See [Sessions](./sessions.md). |
| `admin.api.cookies.sameSite` | `Strict` \| `Lax` \| `None` | `Strict` | `SameSite` attribute for the same cookies. See [Sessions](./sessions.md). |
| `admin.api.origins.allowedOrigins` | array of strings | none | Browser origins allowed to call the Admin API. See [CORS](./cors.md). |
| `admin.api.origins.allowAll` | boolean | `false` | Allow any origin. Cannot be combined with `allowedOrigins` or `denyAll`. See [CORS](./cors.md). |
| `admin.api.origins.denyAll` | boolean | `false` | Reject every request that has an `Origin` header. See [CORS](./cors.md). |
| `admin.api.tls.serverCertFile` | string | none | PEM server certificate for the admin port. Required if `tls` is set. See [TLS](./tls.md). |
| `admin.api.tls.serverKeyFile` | string | none | PEM private key for the admin port. Required if `tls` is set. See [TLS](./tls.md). |

The authoritative schema is [`internal/config/schemas/admin_schema.json`](../internal/config/schemas/admin_schema.json). The [Configuration](./configuration.md) page explains `${env:...}` references and how validation errors are reported.

## The superuser

### Bootstrap

When Kerberos starts and its database has no superuser, it creates one from `admin.superUser.clientId` and `admin.superUser.clientSecret`. If you leave those keys out, the superuser is `admin` / `secret`.

After that, **Kerberos ignores these settings.** Changing `clientId` or `clientSecret` in the configuration and restarting does not change the superuser's name or password. The values in the database are used from then on. To get new bootstrap values applied, you would have to start with an empty database.

### Change the password immediately

The default credentials are public. As soon as Kerberos has started for the first time, log in as the superuser and change the password:

```bash
curl -i -c su.jar -b su.jar -X POST http://localhost:30001/api/admin/superuser/login \
  -H 'Content-Type: application/json' \
  -d '{"clientId": "admin", "clientSecret": "secret"}'

curl -i -c su.jar -b su.jar -X PUT http://localhost:30001/api/admin/superuser/password \
  -H 'Content-Type: application/json' \
  -d '{"oldPassword": "secret", "newPassword": "a-long-random-passphrase"}'
```

A successful change returns `204`. A wrong `oldPassword` returns `400`. Changing the password does not end superuser sessions that already exist. The superuser's name (`clientId`) cannot be changed through the API.

### What the superuser can do

- The superuser holds **every permission**, including permissions added to Kerberos in the future. Permission checks always pass for the superuser.
- It logs in, refreshes and logs out through its own endpoints: `/api/admin/superuser/login`, `/api/admin/superuser/refresh` and `/api/admin/superuser/logout`. It cannot use `/api/admin/login`.
- `POST /api/admin/superuser/logout` ends **all** superuser sessions, not only the one making the request. Anyone else logged in as the superuser is logged out too.
- The superuser does not appear in `GET /api/admin/users`, and you cannot fetch, rename, delete or regroup it through the `/api/admin/users/...` endpoints.
- `GET /api/admin/me` returns `{"isSuperuser": true}` with no `user` object for a superuser session.
- On the basic-auth API, a superuser session has full access to every organisation. See [Organisations](./organizations.md).

Use the superuser to set up admin groups and admin users, then use those accounts for day-to-day work.

## Admin users and admin groups

An **admin user** has a username and a password. Usernames are unique, and they share one namespace with the superuser: you cannot create an admin user with the superuser's name (`409`). The API requires only that a password is non-empty. Kerberos does not enforce a length or strength policy for admin users, so set your own.

An **admin group** has a unique name and a set of permissions. Groups are the only way to give an admin user permissions. A user with no groups can log in, read `/api/admin/me` and `/api/admin/permissions`, and manage their own account, but nothing else.

### Permissions

The set of permissions is fixed. Their IDs are stable and are the same on every installation. `GET /api/admin/permissions` lists them.

| ID | Name | Grants |
| --- | --- | --- |
| 1 | `flow-viewer` | `GET /api/admin/flow`: view the active request flow. |
| 2 | `oas-viewer` | `GET /api/admin/oas/{backend}`: download a backend's OpenAPI specification. |
| 3 | `basic-auth-org-admin` | Full access to every organisation through the basic-auth API, including creating and renaming organisations. See [Organisations](./organizations.md). |
| 4 | `basic-auth-org-viewer` | Read-only (`GET`) access to every organisation through the basic-auth API. |
| 5 | `admin-user-mgmt-admin` | Create, update and delete admin users and admin groups; set users' group memberships; change other users' passwords. Includes everything `admin-user-mgmt-viewer` allows. |
| 6 | `admin-user-mgmt-viewer` | List and read admin users and admin groups. |
| 7 | `debugger` | All `/api/admin/debug/...` endpoints. See [Admin debugging](./admin-debugging.md). |

### How effective permissions are computed

- A user's permissions are the **union** of the permissions of all groups they belong to. There are no deny rules.
- Permissions are looked up **on every request**, not when the user logs in. Adding a user to a group, removing them from one, changing a group's permissions or deleting a group takes effect on the user's next request. The user does not need to log in again.
- Every admin user can read and rename their own account and change their own password, whatever their permissions.
- Deleting an admin user ends all of that user's sessions straight away. Changing a user's password does not end their sessions.
- Deleting a group removes its memberships. Its members lose the permissions it granted.

## Behaviour

### Request processing

Kerberos checks each request to `/api/admin/...` in this order. The first check that fails decides the response:

1. **Origin (CORS).** A disallowed `Origin` returns `403`. See [CORS](./cors.md).
2. **Request validation.** The request is checked against [`admin.yaml`](../openapi/admin.yaml): required body fields, field types, empty strings where `minLength: 1` applies, integer path parameters, and so on. A failure returns `400`. For example, `GET /api/admin/users/abc` returns `400` because `userID` must be an integer.
3. **CSRF.** A non-`GET` request that has an `Origin` header must send a matching `X-Krb-Csrf-Token` header, or it gets `403`. The two login endpoints are exempt. Requests without an `Origin` header, such as those from curl or server-side clients, are not checked. See [Sessions](./sessions.md).
4. **Session.** Every endpoint except `POST /superuser/login`, `/login`, `/superuser/refresh` and `/refresh` needs a valid, unexpired `session` cookie, or it gets `401`.
5. **Permission.** A logged-in caller without the required permission gets `403`.

Errors have this body:

```json
{"errors": ["Forbidden"]}
```

### Sessions

- An admin session lasts **15 minutes**.
- A refresh replaces the session and refresh token and issues new cookies. It is accepted from login until 45 minutes after the session expires, which is one hour after login. After that window, log in again.
- Admin users refresh at `POST /api/admin/refresh`. The superuser refreshes at `POST /api/admin/superuser/refresh`. The `refresh` cookie is scoped to that path.
- `POST /api/admin/logout` ends only the current session.

Cookie attributes, the CSRF double-submit flow, and the cookie-name clash with the basic-auth API (both use a cookie named `session`) are covered in [Sessions](./sessions.md).

## Endpoint reference

All paths are relative to `/api/admin` on the admin port. "Any session" means any logged-in admin user or the superuser. `401` (no valid session) and `400` (request validation) can occur on any endpoint marked as needing a session, and `500` on any endpoint. Those codes are not repeated below.

### Superuser

| Method and path | Who | Body | Notable responses |
| --- | --- | --- | --- |
| `POST /superuser/login` | public | `{"clientId", "clientSecret"}` | `204` + cookies; `401` wrong credentials |
| `POST /superuser/logout` | superuser | none | `204` ends all superuser sessions; `403` if caller is not the superuser |
| `POST /superuser/refresh` | superuser refresh cookie | none | `204` + new cookies; `401` missing, unknown or too old refresh token; `403` token belongs to an admin user |
| `PUT /superuser/password` | superuser | `{"oldPassword", "newPassword"}` | `204`; `400` wrong old password; `403` not the superuser |

### Admin-user session

| Method and path | Who | Body | Notable responses |
| --- | --- | --- | --- |
| `POST /login` | public | `{"username", "password"}` | `204` + cookies; `401` unknown user or wrong password |
| `POST /logout` | any session | none | `204` ends the current session |
| `POST /refresh` | admin-user refresh cookie | none | `204` + new cookies; `401` missing, unknown or too old refresh token |

### Me and permissions

| Method and path | Who | Notable responses |
| --- | --- | --- |
| `GET /me` | any session | `200` `{"isSuperuser": bool, "user": {"id", "username", "groups": [{"id", "name"}]}}`; `user` is omitted for the superuser |
| `GET /permissions` | any session | `200` list of `{"id", "name"}` |

### Admin users

| Method and path | Required permission | Body | Notable responses |
| --- | --- | --- | --- |
| `POST /users` | `admin-user-mgmt-admin` | `{"username", "password"}` | `201` `{"id", "username"}`; `409` username taken |
| `GET /users` | `admin-user-mgmt-admin` or `-viewer` | none | `200` list of users, each with `groups` |
| `GET /users/{userID}` | self, or `admin-user-mgmt-admin` / `-viewer` | none | `200` user with `groups`; `404` |
| `PUT /users/{userID}` | self, or `admin-user-mgmt-admin` | `{"username"}` | `204`; `409` username taken |
| `DELETE /users/{userID}` | `admin-user-mgmt-admin` | none | `204`; `404` |
| `PUT /users/{userID}/password` | self, or `admin-user-mgmt-admin` | `{"oldPassword", "newPassword"}` | `204`; `400` wrong old password; `404` |
| `PUT /users/{userID}/groups` | `admin-user-mgmt-admin` | `{"groupIDs": [int]}` | `204`; `404` user not found |

`PUT /users/{userID}/groups` sets the **complete** list of groups. Groups that are not in `groupIDs` are removed, and `{"groupIDs": []}` removes the user from every group.

`PUT /users/{userID}/password` always requires the user's current password, even when an `admin-user-mgmt-admin` changes someone else's password. Administrators cannot reset a forgotten password. Delete and re-create the user instead.

### Admin groups

| Method and path | Required permission | Body | Notable responses |
| --- | --- | --- | --- |
| `POST /groups` | `admin-user-mgmt-admin` | `{"name", "permissionIDs": [int]}` | `201` `{"id", "name", "permissions": [{"id", "name"}]}`; `409` name taken |
| `GET /groups` | `admin-user-mgmt-admin` or `-viewer` | none | `200` list of groups, each with `permissions` |
| `GET /groups/{groupID}` | `admin-user-mgmt-admin` or `-viewer` | none | `200` group with `permissions`; `404` |
| `PUT /groups/{groupID}` | `admin-user-mgmt-admin` | `{"name", "permissionIDs": [int]}` | `204`; `404`; `409` name taken |
| `DELETE /groups/{groupID}` | `admin-user-mgmt-admin` | none | `204`; `404` |

`permissionIDs` is required on both create and update. On update it **replaces** the group's permissions, so always send the full set you want. Use only IDs from the [permissions table](#permissions), with no duplicates. An unknown or repeated ID makes the request fail with `500` (see [Things to know](#things-to-know)).

### Flow and OAS

| Method and path | Required permission | Notable responses |
| --- | --- | --- |
| `GET /flow` | `flow-viewer` | `200` list of flow components (see [Inspecting the flow](#inspecting-the-flow)) |
| `GET /oas/{backend}` | `oas-viewer` | `200` specification file (`application/yaml`); `404` backend has no OAS mapping |

### Debug

All debug endpoints require the `debugger` permission. They are documented in [Admin debugging](./admin-debugging.md).

| Method and path | Purpose |
| --- | --- |
| `POST /debug/{backend}/sessions`, `GET /debug/{backend}/sessions` | Start or list debug sessions for a backend |
| `GET`, `PUT`, `POST`, `DELETE /debug/{backend}/sessions/{sessionId}` | Read, extend, stop or delete a debug session |
| `GET /debug/{backend}/sessions/{sessionId}/calls`, `.../calls/{callId}` | Inspect captured calls |

## Walkthrough: give a colleague access

This walkthrough creates an `operators` group that can view the flow and OpenAPI specifications and run debug sessions, then creates an admin user `alice` in that group. The curl commands use a cookie jar file to store the session cookies.

> The cookies carry the `Secure` attribute. curl and browsers send them over plain HTTP to `localhost`, but for any other host you need HTTPS on the admin port. See [TLS](./tls.md).

**1. Log in as the superuser.**

```bash
curl -i -c su.jar -b su.jar -X POST http://localhost:30001/api/admin/superuser/login \
  -H 'Content-Type: application/json' \
  -d '{"clientId": "admin", "clientSecret": "a-long-random-passphrase"}'
# HTTP/1.1 204 No Content
# Set-Cookie: session=...; Set-Cookie: refresh=...; Set-Cookie: csrf=...
```

**2. Create a group with permissions** `flow-viewer` (1), `oas-viewer` (2) and `debugger` (7).

```bash
curl -s -c su.jar -b su.jar -X POST http://localhost:30001/api/admin/groups \
  -H 'Content-Type: application/json' \
  -d '{"name": "operators", "permissionIDs": [1, 2, 7]}'
```

```json
{"id": 1, "name": "operators", "permissions": [{"id": 1, "name": "flow-viewer"}, {"id": 2, "name": "oas-viewer"}, {"id": 7, "name": "debugger"}]}
```

**3. Create the admin user.**

```bash
curl -s -c su.jar -b su.jar -X POST http://localhost:30001/api/admin/users \
  -H 'Content-Type: application/json' \
  -d '{"username": "alice", "password": "initial-password-123"}'
```

```json
{"id": 2, "username": "alice"}
```

**4. Add the user to the group.** Use the IDs returned above.

```bash
curl -i -c su.jar -b su.jar -X PUT http://localhost:30001/api/admin/users/2/groups \
  -H 'Content-Type: application/json' \
  -d '{"groupIDs": [1]}'
# HTTP/1.1 204 No Content
```

**5. Log in as the user** with a separate cookie jar.

```bash
curl -i -c alice.jar -b alice.jar -X POST http://localhost:30001/api/admin/login \
  -H 'Content-Type: application/json' \
  -d '{"username": "alice", "password": "initial-password-123"}'
# HTTP/1.1 204 No Content
```

**6. Check who you are.**

```bash
curl -s -c alice.jar -b alice.jar http://localhost:30001/api/admin/me
```

```json
{"isSuperuser": false, "user": {"id": 2, "username": "alice", "groups": [{"id": 1, "name": "operators"}]}}
```

Alice can now call `GET /api/admin/flow`. A call to `GET /api/admin/users` returns `403` because she lacks `admin-user-mgmt-viewer`. She should change her password straight away with `PUT /api/admin/users/2/password`.

## Inspecting the flow

`GET /api/admin/flow` (permission `flow-viewer`) describes the components that every gateway request passes through, in order. Each entry has a `name` and a `data` object whose content depends on the component:

| `name` | `data` |
| --- | --- |
| `obs` | `enabled`: whether observability is on. See [Observability](./observability.md). |
| `router` | `backends`: each configured backend's `name`, `host` and `port`. See [Routing](./routing.md). |
| `authorizer` | Present only when the `auth` section is configured. `methods` (for example `basic`) and `scheme.mappings` with each mapping's `backend`, `method`, `exempt` patterns and `authorization` groups and paths. See [Authentication](./authentication.md). |
| `oas-validator` | Present only when the `oas` section is configured. `backends`: backends with request validation. See [Request validation](./request-validation.md). |
| `forwarder` | Empty; forwards the request to the backend. |

`authorizer` and `oas-validator` appear in the order set by their `order` settings. Example:

```json
[
  {"name": "obs", "data": {"enabled": true}},
  {"name": "router", "data": {"backends": [{"name": "echo", "host": "echo", "port": 8080}]}},
  {"name": "authorizer", "data": {"methods": {"basic": {}}, "scheme": {"mappings": [{"backend": "echo", "method": "basic", "exempt": ["/public/*"], "authorization": {"groups": ["staff"]}}]}}},
  {"name": "oas-validator", "data": {"backends": ["echo"]}},
  {"name": "forwarder", "data": {}}
]
```

The response reflects the configuration Kerberos loaded at startup. Secrets such as TLS key paths and database credentials are not included.

## Inspecting a backend's OpenAPI specification

`GET /api/admin/oas/{backend}` (permission `oas-viewer`) returns the specification file configured for that backend in the `oas` section:

```bash
curl -s -b alice.jar http://localhost:30001/api/admin/oas/echo
```

- The file is read from disk on each request and returned as-is, with `Content-Type: application/yaml`, even if the file is JSON. If the file has changed since startup, you see the new content, but validation still uses the version loaded at startup.
- The full file is returned even if `validateBody` is turned off for the backend.
- A backend with no OAS mapping, or a deployment with no `oas` section, returns `404`.

## Admin access to the basic-auth API

Admin sessions also work on the basic-auth API (`/api/auth/basic/...`, also on the admin port). This lets platform operators manage organisations without having an organisation account:

- The superuser and users with `basic-auth-org-admin` have full access to every organisation. Only admin sessions can create, list or rename organisations.
- Users with `basic-auth-org-viewer` can make `GET` requests on every organisation. Other methods return `403`.

Both APIs use a cookie named `session`. If one browser logs in to both APIs on the same host, the second login replaces the first cookie. [Sessions](./sessions.md) describes how to avoid this, and [Organisations](./organizations.md) describes the basic-auth API itself.

## Go client

Go programs can use the generated client in `github.com/trebent/kerberos/pkg/admin/client`, built from [`admin.yaml`](../openapi/admin.yaml). It provides `NewClient` and `NewClientWithResponses`, which returns typed response bodies, along with the options `WithHTTPClient`, `WithRequestEditorFn` and `WithBaseURL`. Session cookies are not handled for you: pass an `http.Client` with a cookie jar.

```go
package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"net/http/cookiejar"

	adminclient "github.com/trebent/kerberos/pkg/admin/client"
)

func main() {
	ctx := context.Background()
	jar, _ := cookiejar.New(nil)

	c, err := adminclient.NewClientWithResponses(
		"https://kerberos.example.com:30001",
		adminclient.WithHTTPClient(&http.Client{Jar: jar}),
	)
	if err != nil {
		log.Fatal(err)
	}

	login, err := c.LoginWithResponse(ctx, adminclient.LoginJSONRequestBody{
		Username: "alice",
		Password: "her-password",
	})
	if err != nil || login.StatusCode() != http.StatusNoContent {
		log.Fatalf("login failed: %v %v", err, login)
	}

	me, err := c.GetMeWithResponse(ctx)
	if err != nil || me.JSON200 == nil {
		log.Fatalf("me failed: %v", err)
	}
	fmt.Println("logged in as", me.JSON200.User.Username)
}
```

The standard Go cookie jar sends `Secure` cookies over plain `http://` only to `localhost`, so use HTTPS on the admin port when the client talks to any other host. Server-side clients do not send an `Origin` header, so no CSRF header is needed.

## Things to know

- **Bootstrap credentials apply once.** Editing `admin.superUser` after the first start has no effect. Change the password with `PUT /api/admin/superuser/password`.
- **Superuser logout ends every superuser session.**
- **Password changes do not end sessions**, for both the superuser and admin users. Deleting an admin user does end that user's sessions.
- **Admins cannot reset another user's password** without knowing the old one.
- **Renaming a user that does not exist** (`PUT /users/{userID}` with an unknown ID) returns `204` and changes nothing.
- **Group writes are not all-or-nothing.** An unknown or duplicate ID in `permissionIDs` returns `500`. On `POST /groups` the group has already been created at that point, with no permissions. On `PUT /groups/{groupID}` the new name has already been saved. Check the result with `GET /groups` and correct it. `PUT /users/{userID}/groups` with an unknown group ID also returns `500`, but in that case nothing changes.
- **No password policy.** Admin user and superuser passwords only have to be non-empty.
- **Sessions are stored in the database**, so a restart keeps them, and replicas that share a database (PostgreSQL) share them. Debug-session capture state is held in memory. See [Admin debugging](./admin-debugging.md).

## Related docs

- [Sessions](./sessions.md): cookies, lifetimes, refresh, CSRF.
- [CORS](./cors.md): `admin.api.origins`.
- [TLS](./tls.md): `admin.api.tls`.
- [Admin debugging](./admin-debugging.md): debug sessions.
- [Organisations](./organizations.md): the basic-auth API that admin sessions can also manage.
- [Admin connector](./admin-connector.md): protect internal tools behind the admin login.
- [Deployment](./deployment.md): ports, `OAS_DIRECTORY`, security checklist.
- [`../openapi/admin.yaml`](../openapi/admin.yaml): the API contract. A rendered reference is published from `pages/index.html`.
