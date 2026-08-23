# Organisations

Organisations are the tenant boundary in Kerberos authentication.

Each organisation isolates:

- users
- groups
- user-to-group bindings
- sessions

## High-level model

- Users and groups are always organisation-scoped.
- Authorization is based on user group membership.
- Organisation data and sessions are isolated across tenants.

## Organisation administration

Organisation administration is provided by the basic auth API.

At a high level, this includes:

- organisation lifecycle operations
- user lifecycle operations inside an organisation
- group lifecycle operations inside an organisation
- managing user group bindings

## Organisation administrators

Kerberos supports organisation administrators with elevated permissions inside their own organisation.

Typical capabilities include managing users, groups, and memberships for that organisation.

## Relationship to platform admin

Organisation administration and platform administration are separate concerns:

- Organisation administration is tenant-scoped (basic auth model).
- Platform administration uses the Admin API and its own admin user/superuser model.

See [Admin API](./admin-api.md) for platform-level administration.

## API contract

For exact endpoint definitions and schemas, use [`openapi/auth_basic.yaml`](../openapi/auth_basic.yaml).
