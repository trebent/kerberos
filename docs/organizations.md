# Organisations, users and groups

Basic authentication in Kerberos is multi-tenant: every user and group belongs to exactly one
organisation, and every login session is tied to it. This page explains that model, who is allowed
to do what, and how to manage it through the basic-auth API, either with `curl` or with the Go
client `pkg/auth/client`.

The basic-auth API exists only when `auth.methods.basic` is configured. It is served on the
**admin port** (30001 in these examples), not on the gateway port, under the prefix
`/api/auth/basic`. The authoritative contract is [`openapi/auth_basic.yaml`](../openapi/auth_basic.yaml).

## The model

```text
organisation
 ├── users            (name unique within the organisation)
 │     └── sessions   (one per login)
 ├── groups           (name unique within the organisation)
 └── group bindings   (user <-> group membership)
```

| Entity | What it is | Rules |
| --- | --- | --- |
| Organisation | A tenant. Has a numeric `id` and a `name`. | Names are unique across Kerberos. |
| User | Someone who can log in to one organisation. | Names are unique within the organisation (the same name can exist in two organisations). Created with a name of at least 5 characters and a password. |
| Group | A named set of users, used for access rules on the gateway. | Names are unique within the organisation. |
| Group binding | Membership of one user in one group. | Managed per user, as a complete list (see [Assign groups](#6-assign-groups-to-the-user)). |
| Session | Created by a successful login, carried in cookies. | Belongs to one user in one organisation. See [Sessions](./sessions.md). |

Deletes cascade:

- Deleting an organisation deletes all its users, groups, group bindings and sessions.
- Deleting a user deletes the user's group bindings and sessions.
- Deleting a group deletes its group bindings (users lose that membership).

Everything is identified by numeric IDs in the API. A user logs in to a specific organisation by its
ID (`POST /api/auth/basic/organisations/{orgID}/login`), so users must be told their organisation ID.

On the gateway, group **names** are what matter: access rules in the `auth` section list group names,
and an authenticated request carries the user's organisation and user IDs (and, when a group check
runs, group names) to the backend. See [Authentication](./authentication.md).

## Roles

There are three kinds of caller.

| Role | Who | How they authenticate |
| --- | --- | --- |
| Platform administrator | The superuser, and admin users whose groups grant the `basic-auth-org-admin` or `basic-auth-org-viewer` permission. These are Admin API accounts, not basic-auth users. | Log in through the Admin API (`/api/admin/...`) and call the basic-auth API with that admin session cookie. See [Admin API](./admin-api.md). |
| Organisation administrator | The user `admin-<orgname>` that Kerberos creates together with each organisation. | Logs in to its organisation like any other user. |
| Regular user | Every user created through the API. | Logs in to its organisation. |

### Platform administrators

- The superuser and holders of `basic-auth-org-admin` have full access to every endpoint, for every
  organisation.
- Holders of `basic-auth-org-viewer` (without `basic-auth-org-admin`) can call every `GET` endpoint
  for every organisation; any other method returns `403`.
- Platform administrators are the only callers that can create, list or rename organisations.

### The organisation administrator

When an organisation is created, Kerberos also creates its administrator in the same step:

- Username: `admin-<orgname>`, for example `admin-acme` for organisation `acme`.
- Password: randomly generated (a 36-character UUID string) and returned **once**, in the
  `adminPassword` field of the create response. It is stored only as a hash and cannot be retrieved
  later. Store it securely straight away.

This is the only user with administrator rights in the organisation. The API has no way to create
another administrator or to promote a user. If the administrator account is deleted, only platform
administrators can manage the organisation afterwards.

Renaming the organisation does not rename the administrator: it keeps its original
`admin-<oldname>` username.

### Regular users

Regular users can see and manage only themselves: view and rename their own user, change their own
password, list their own groups, delete themselves, and log out.

## Who can do what

All rights are scoped to the caller's own organisation. A basic-auth session used against another
organisation's ID gets `403`, whatever the role.

| Action | Platform admin (`basic-auth-org-admin` or superuser) | Platform viewer (`basic-auth-org-viewer`) | Organisation administrator | Regular user |
| --- | --- | --- | --- | --- |
| Create an organisation | yes | no | no | no |
| List organisations | yes | yes | no | no |
| Rename an organisation | yes | no | no | no |
| Get an organisation | yes | yes | own | no |
| Delete an organisation | yes | no | own | no |
| Create / list users | yes | list only | yes | no |
| Get a user | yes | yes | any user | self |
| Rename a user | yes | no | any user | self |
| Delete a user | yes | no | any user | self |
| Change a user's password (old password required) | yes | no | any user | self |
| View a user's groups | yes | yes | any user | self |
| Set a user's groups | yes | no | yes | no |
| Create / list / get / rename / delete groups | yes | get and list only | yes | no |
| `GET /users/me` | no (see [Things to know](#things-to-know)) | no | yes | yes |
| Log out | no (use the Admin API logout) | no | yes | yes |

## Endpoint reference

All paths are relative to the admin port and start with `/api/auth/basic`. `{orgID}`, `{userID}`
and `{groupID}` are numeric; a non-numeric value returns `400`.

"Org admin" below means the organisation administrator of `{orgID}`; platform administrators with
`basic-auth-org-admin` (or the superuser) can call everything, and `basic-auth-org-viewer` can call
every `GET`.

| Method | Path | Who | Request body | Success | Notes |
| --- | --- | --- | --- | --- | --- |
| `POST` | `/organisations` | Platform admin only | `{"name"}` | `201` `{id, name, adminUserId, adminUsername, adminPassword}` | `409` if the name exists. |
| `GET` | `/organisations` | Platform admin / viewer only | | `200` `[{id, name}]` | |
| `GET` | `/organisations/{orgID}` | Org admin | | `200` `{id, name}` | `404` if missing. |
| `PUT` | `/organisations/{orgID}` | Platform admin only | `{"id", "name"}` | `200` `{id, name}` | Renames. The path ID is used; `id` in the body is required but ignored. `404`, `409`. |
| `DELETE` | `/organisations/{orgID}` | Org admin | | `204` | Cascades to all users, groups and sessions. |
| `POST` | `/organisations/{orgID}/login` | Anyone | `{"username", "password"}` | `204` + cookies | `401` on unknown user or wrong password. |
| `POST` | `/organisations/{orgID}/logout` | Any user of the organisation | | `204` | Ends the current session and expires its cookies. |
| `POST` | `/organisations/{orgID}/refresh` | Holder of a refresh cookie | | `204` + new cookies | `401` if the refresh cookie is missing, unknown or too late. See [Sessions](./sessions.md). |
| `POST` | `/organisations/{orgID}/users` | Org admin | `{"name", "password"}` | `201` `{id, name}` | Name at least 5 characters; password 10 to 40 characters. `409` if the name exists. |
| `GET` | `/organisations/{orgID}/users` | Org admin | | `200` `[{id, name, groups}]` | Each user includes its groups. |
| `GET` | `/organisations/{orgID}/users/me` | Any user of the organisation | | `200` `{id, name, groups}` | The logged-in user. |
| `GET` | `/organisations/{orgID}/users/{userID}` | Org admin or the user | | `200` `{id, name, groups}` | `404` if missing. |
| `PUT` | `/organisations/{orgID}/users/{userID}` | Org admin or the user | `{"id", "name"}` | `200` `{id, name}` | Renames. `id` in the body is required but ignored. `404`, `409`. |
| `DELETE` | `/organisations/{orgID}/users/{userID}` | Org admin or the user | | `204` | |
| `PUT` | `/organisations/{orgID}/users/{userID}/password` | Org admin or the user | `{"oldPassword", "password"}` | `204` | Old password is always required. `401` if it is wrong. |
| `GET` | `/organisations/{orgID}/users/{userID}/groups` | Org admin or the user | | `200` `[{id, name}]` | |
| `PUT` | `/organisations/{orgID}/users/{userID}/groups` | Org admin | `[{"id", "name"}, ...]` | `200` | Desired state: replaces all memberships. |
| `POST` | `/organisations/{orgID}/groups` | Org admin | `{"name"}` | `201` `{id, name}` | `409` if the name exists. |
| `GET` | `/organisations/{orgID}/groups` | Org admin | | `200` `[{id, name}]` | |
| `GET` | `/organisations/{orgID}/groups/{groupID}` | Org admin | | `200` `{id, name}` | `404` if missing. |
| `PUT` | `/organisations/{orgID}/groups/{groupID}` | Org admin | `{"id", "name"}` | `200` `{id, name}` | Renames. `id` in the body is required but ignored. `404`, `409`. |
| `DELETE` | `/organisations/{orgID}/groups/{groupID}` | Org admin | | `204` | Removes the group from all its members. |

Every endpoint except login needs a valid session cookie; without one the response is `401`.
Requests are validated against the OpenAPI contract before they reach the handler, so a missing
field or a too-short name returns `400`.

## Walkthrough

This example sets up an organisation `acme` with a group `editors` and a user `alice`. It uses
separate `curl` cookie jars per person, which also avoids the
[cookie-name clash](./sessions.md) between Admin API and basic-auth sessions.

The session cookies are marked `Secure`. `curl` and browsers send them over plain HTTP only to
`localhost`; anywhere else, serve the admin port over HTTPS (see [TLS](./tls.md)).

### Prerequisite: enable basic authentication

A minimal `auth` section (the `scheme` block is required together with `methods`):

```json
{
  "auth": {
    "methods": {
      "basic": {}
    },
    "scheme": {
      "mappings": [
        {
          "backend": "echo",
          "method": "basic",
          "authorization": { "groups": ["editors"] }
        }
      ]
    }
  }
}
```

See [Authentication](./authentication.md) for mappings and group rules, and
[Configuration](./configuration.md) for the full file.

### 1. Platform administrator logs in

Log in as the superuser (or an admin user holding `basic-auth-org-admin`) through the Admin API:

```bash
curl -i -c platform.jar -X POST http://localhost:30001/api/admin/superuser/login \
  -H 'Content-Type: application/json' \
  -d '{"clientId": "admin", "clientSecret": "secret"}'
```

### 2. Create the organisation

```bash
curl -s -b platform.jar -X POST http://localhost:30001/api/auth/basic/organisations \
  -H 'Content-Type: application/json' \
  -d '{"name": "acme"}'
```

```json
{
  "id": 1,
  "name": "acme",
  "adminUserId": 1,
  "adminUsername": "admin-acme",
  "adminPassword": "3f0c6a8e-6f1e-4d0b-9a57-2c1d8f4e7b21"
}
```

Save `id` and `adminPassword` now. The password is never shown again. Hand both, together with the
username, to whoever runs the organisation.

### 3. Organisation administrator logs in

```bash
curl -i -c acme-admin.jar -X POST http://localhost:30001/api/auth/basic/organisations/1/login \
  -H 'Content-Type: application/json' \
  -d '{"username": "admin-acme", "password": "3f0c6a8e-6f1e-4d0b-9a57-2c1d8f4e7b21"}'
```

The response is `204 No Content` with `session`, `refresh` and `csrf` cookies. The session lasts
15 minutes; see [Sessions](./sessions.md) for refreshing it.

The administrator should change the generated password (old password required):

```bash
curl -i -b acme-admin.jar -X PUT \
  http://localhost:30001/api/auth/basic/organisations/1/users/1/password \
  -H 'Content-Type: application/json' \
  -d '{"oldPassword": "3f0c6a8e-6f1e-4d0b-9a57-2c1d8f4e7b21", "password": "a-new-strong-pass"}'
```

### 4. Create a group

```bash
curl -s -b acme-admin.jar -X POST http://localhost:30001/api/auth/basic/organisations/1/groups \
  -H 'Content-Type: application/json' \
  -d '{"name": "editors"}'
```

```json
{ "id": 1, "name": "editors" }
```

### 5. Create a user

```bash
curl -s -b acme-admin.jar -X POST http://localhost:30001/api/auth/basic/organisations/1/users \
  -H 'Content-Type: application/json' \
  -d '{"name": "alice", "password": "alice-password-1"}'
```

```json
{ "id": 2, "name": "alice" }
```

### 6. Assign groups to the user

`PUT .../users/{userID}/groups` takes the **complete** list of groups the user should be in. Groups
not in the list are removed, groups in the list are added, and an empty list `[]` removes all
memberships. Each entry needs `id` and `name`, but only `id` is used.

```bash
curl -s -b acme-admin.jar -X PUT \
  http://localhost:30001/api/auth/basic/organisations/1/users/2/groups \
  -H 'Content-Type: application/json' \
  -d '[{"id": 1, "name": "editors"}]'
```

The response repeats the list you sent. To see the stored memberships, use
`GET .../users/2/groups`.

### 7. The user logs in

```bash
curl -i -c alice.jar -X POST http://localhost:30001/api/auth/basic/organisations/1/login \
  -H 'Content-Type: application/json' \
  -d '{"username": "alice", "password": "alice-password-1"}'

curl -s -b alice.jar http://localhost:30001/api/auth/basic/organisations/1/users/me
```

```json
{ "id": 2, "name": "alice", "groups": [{ "id": 1, "name": "editors" }] }
```

The same `session` cookie now authenticates Alice on the gateway port, for example
`curl -b alice.jar http://localhost:30000/gw/backend/echo/hello`.

## Passwords

- New users need a password of **10 to 40 characters**.
- A password change needs the current password in `oldPassword`, even for the organisation
  administrator and platform administrators. There is no reset: if a user forgets their password,
  delete the user and create it again (group memberships must then be reassigned).
- Use **ASCII-only** passwords, and keep changed passwords to **40 characters or fewer**. Passwords
  containing non-ASCII characters (accented letters, emoji and so on) are accepted when set but the
  user can then not log in with them. A new password longer than 40 bytes is not rejected by
  validation on the password-change endpoint, but the request fails.
- Changing a password does not end the user's existing sessions.
- Passwords are stored as salted bcrypt hashes.

## Errors

Errors have a JSON body of the form `{"errors": ["<message>"]}`, except `404`, which has no body.

| Status | When |
| --- | --- |
| `400` | The request does not match the OpenAPI contract: missing or invalid fields, a name or password that is too short, a non-numeric ID. |
| `401` | No session cookie, an unknown or expired session, wrong username or password at login, wrong `oldPassword`, or an unusable refresh cookie. |
| `403` | The session belongs to another organisation, the caller lacks the role (for example a regular user listing users, an organisation user creating an organisation, a viewer sending a non-`GET`), or a browser request failed the CSRF check (see [Sessions](./sessions.md)). |
| `404` | The organisation, user or group does not exist (get and rename only). Also returned, with an error body, when an organisation user calls `PUT /organisations/{orgID}`, which only platform administrators can use. |
| `409` | The organisation, user or group name is already taken in its scope. |
| `500` | Unexpected server error, including the binding cases listed below. |

## Go client

`github.com/trebent/kerberos/pkg/auth/client` is a client generated from
[`openapi/auth_basic.yaml`](../openapi/auth_basic.yaml). It has one method per operation, plus
`...WithResponse` variants that decode the JSON body into `JSON200`, `JSON201`, `JSON401` and so on.

The client does not manage cookies itself. Give it an `http.Client` with a cookie jar so the login
cookies are sent on later calls. Requests from the client carry no `Origin` header, so no CSRF token
is needed.

```go
package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"net/http/cookiejar"

	authclient "github.com/trebent/kerberos/pkg/auth/client"
)

func main() {
	ctx := context.Background()

	jar, _ := cookiejar.New(nil)
	c, err := authclient.NewClientWithResponses(
		"http://localhost:30001",
		authclient.WithHTTPClient(&http.Client{Jar: jar}),
	)
	if err != nil {
		log.Fatal(err)
	}

	orgID := authclient.Orgid(1)

	login, err := c.LoginWithResponse(ctx, orgID, authclient.LoginJSONRequestBody{
		Username: "alice",
		Password: "alice-password-1",
	})
	if err != nil {
		log.Fatal(err)
	}
	if login.StatusCode() != http.StatusNoContent {
		log.Fatalf("login failed: %s", login.Status())
	}

	me, err := c.GetMeWithResponse(ctx, orgID)
	if err != nil {
		log.Fatal(err)
	}
	if me.JSON200 == nil {
		log.Fatalf("get me failed: %s", me.Status())
	}
	fmt.Println(me.JSON200.Id, me.JSON200.Name)
}
```

Go's cookie jar sends `Secure` cookies over plain HTTP only to `localhost`; for other hosts use an
`https://` base URL.

## Things to know

- **Platform administrators are not organisation users.** They have no user record in the
  organisation, so `GET .../users/me` and `POST .../logout` do not work with an Admin API session
  (the connection is dropped without a normal response). Use `GET .../users/{userID}` instead, and
  log out through the Admin API.
- **Deletes are idempotent.** Deleting an organisation, user or group that does not exist returns
  `204`, not `404`.
- **Lookups of missing parents return empty results.** Listing users or groups of a non-existent
  organisation, or the groups of a non-existent user, returns `200` with `[]`.
- **Group assignment pitfalls.** In `PUT .../users/{userID}/groups`:
  - a group ID that does not exist returns `500`;
  - listing the same group ID twice returns `500` if the user is not already in that group;
  - a group ID from another organisation is accepted but has no effect: the membership is never
    shown and never matches access rules.
- **Renames are not validated for length.** Creating a user requires a name of at least 5
  characters, but renaming does not enforce that minimum.
- **Name length.** Names are limited to 100 characters in PostgreSQL. Because the administrator is
  named `admin-<orgname>`, keep organisation names to 94 characters or fewer.
- **Any organisation's session authenticates on the gateway.** A logged-in user of any organisation
  passes authentication on a mapped backend. Use group rules to restrict access, and use the
  organisation ID that Kerberos forwards to the backend to separate tenants. See
  [Authentication](./authentication.md).
- **Deleting your own administrator.** The organisation administrator can delete itself or its
  organisation. After deleting itself, only platform administrators can manage the organisation.

## Related docs

- [Sessions](./sessions.md): cookies, lifetimes, refresh, logout and CSRF.
- [Authentication](./authentication.md): protecting backends and using groups in access rules.
- [Admin API](./admin-api.md): platform administrators, the superuser and the
  `basic-auth-org-admin` / `basic-auth-org-viewer` permissions.
- [CORS](./cors.md): `auth.methods.basic.api.origins`.
- [`openapi/auth_basic.yaml`](../openapi/auth_basic.yaml): the API contract.
