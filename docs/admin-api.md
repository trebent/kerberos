# Admin API

The Admin API is the operational API for platform-level management in Kerberos.

It covers:

- admin authentication/session handling
- admin user, group, and permission management
- flow and OAS inspection endpoints
- debug session control and call inspection

Canonical API contract: [`openapi/admin.yaml`](../openapi/admin.yaml)

## Access model

Kerberos has two admin-level access types:

- **Superuser**
  - Configured via `admin.superUser` in Kerberos config.
  - Uses dedicated auth paths under `/api/admin/superuser/*`.
  - Has implicit access to all admin permissions.

- **Admin users**
  - Managed through `/api/admin/users` paths.
  - Auth via `/api/admin/login`, `/api/admin/logout`, `/api/admin/refresh`.
  - Access is determined by permissions inherited from group membership.

## Permissions and groups

Admin permissions are assigned to admin groups, and admin users are assigned to those groups.

High-level permission assignment flow:

1. Create an admin group (`/api/admin/groups`).
2. Attach permission IDs to that group.
3. Add users to groups (`/api/admin/users/{userID}/groups`).
4. Users inherit effective permissions from all assigned groups.

Use `/api/admin/permissions` to retrieve available permissions and their IDs.

## Capability areas (high level)

- **Admin user management**: create/read/update/delete admin users, rotate passwords, assign groups.
- **Admin group management**: create/read/update/delete groups, assign permissions.
- **Flow/OAS visibility**: read active flow metadata and backend OAS when permitted.
- **Debugging**: start/stop/manage debug sessions and inspect captured calls.

## Session model

Admin API authentication is cookie-based. Session validity controls access to protected endpoints. Superuser and regular admin sessions are handled through their respective login/refresh/logout path families.

For exact request/response schemas, status codes, and parameter definitions, use [`openapi/admin.yaml`](../openapi/admin.yaml).
