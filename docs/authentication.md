# Authentication

Kerberos provides backend-scoped authentication and authorization through flow components in the gateway request pipeline.

## Authorizer role

The authorizer component is responsible for:

1. Determining which auth method applies to the routed backend.
2. Enforcing authentication for protected routes.
3. Enforcing authorization rules where configured.
4. Passing valid requests to the next flow component.

## Request handling (high level)

For mapped backends:

1. Resolve backend mapping and auth method.
2. Apply exemption checks for configured paths.
3. Validate session/authentication state.
4. Evaluate authorization rules (group-based).
5. Forward only if checks pass.

## Basic authentication model

Kerberos currently uses cookie-backed sessions for the basic auth model.

At a high level:

- Login creates a session.
- Session cookies are used on subsequent requests.
- Refresh extends active sessions.
- Logout invalidates active sessions.

## Organisation and group authorization

Authorization decisions are group-based within organisations.

- Users belong to organisations.
- Users can be bound to groups.
- Auth mappings can require group membership globally or by path.

See [Organisations](./organizations.md) for the tenant model.

## Admin and superuser context

Organisation administrators have elevated capabilities within their organisation. Platform-level superuser access is handled through the Admin API model.

See [Admin API](./admin-api.md) for platform admin access.

## API contract

For endpoint-level details, request/response schemas, and parameter definitions, use:

- [`openapi/auth_basic.yaml`](../openapi/auth_basic.yaml)
- [`openapi/admin.yaml`](../openapi/admin.yaml)
