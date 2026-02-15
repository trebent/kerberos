# Authentication

Kerberos provides a flexible authentication system that can be configured per backend. The authentication flow is managed by the authorizer component, which acts as middleware in the request processing chain.

## Authorizer

The authorizer (`internal/auth/authorizer.go`) is a flow component that:

1. **Determines the authentication method** for each backend based on configuration
2. **Validates authentication** by checking if the request contains valid credentials
3. **Validates authorization** by verifying if the authenticated user has permission to access the requested resource
4. **Forwards authenticated requests** to the next component in the chain

### Request Flow

The authorizer processes requests in the following order:

1. Extract the backend name from the request context
2. Find the authentication method configured for that backend
3. If no method is configured or the path is exempted, pass the request through without authentication
4. Validate that the user is authenticated (has a valid session)
5. Validate that the user is authorized (has the required group memberships)
6. Forward the request to the next handler if both checks pass

### Path Exemptions

Backends can be configured with path exemptions that bypass authentication. These are specified using glob patterns in the configuration and are useful for public endpoints like health checks or documentation.

## Basic Authentication

Basic authentication is the primary authentication method supported by Kerberos. It uses session-based authentication with the following components:

### Session Management

- Sessions are created upon successful login via the `/auth/basic/{orgID}/login` endpoint
- Each session is identified by a unique session ID returned in the `X-Krb-Session` header
- Sessions have a 15-minute expiration time
- Subsequent requests must include the session ID in the `X-Krb-Session` header
- Users can logout via the `/auth/basic/{orgID}/logout` endpoint, which invalidates all their active sessions

### Authentication Process

1. **Login**: Users provide username, password, and organisation ID
2. **Session Creation**: On successful authentication, a session is created and its ID is returned
3. **Request Authentication**: For each authenticated request, the authorizer:
   - Extracts the session ID from the `X-Krb-Session` header
   - Queries the database to validate the session
   - Checks if the session has expired
   - Adds `X-Krb-Org` and `X-Krb-User` headers to the request with the user's organisation and user IDs

### Authorization Process

Authorization in Kerberos is based on group membership. The authorizer:

1. Checks if the backend has authorization rules configured
2. Determines which groups are required for the requested path (path-specific rules override global rules)
3. Queries the database for the user's group memberships within their organisation
4. Adds all group names to the request via `X-Krb-Groups` headers
5. Grants access if the user belongs to at least one required group

### Authentication API

The basic authentication method exposes a comprehensive REST API for managing:

- **Organisations**: Create, read, update, and delete organisations
- **Users**: Create, read, update, and delete users within organisations
- **Groups**: Create, read, update, and delete groups within organisations
- **Group Bindings**: Assign users to groups
- **Sessions**: Login and logout operations
- **Password Management**: Change user passwords

All API endpoints are scoped to organisations via the `{orgID}` path parameter.

## Administrator Accounts

Administrator accounts are special user accounts with elevated privileges within their organisation.

### Administrator Privileges

An administrator account within an organisation can:

- Create, update, and delete users in their organisation
- Create, update, and delete groups in their organisation
- Manage group bindings (assign users to groups)
- View and manage all organisation details
- Access all auth API paths that require administrator privileges

Administrator accounts are automatically granted access to operations that would normally be restricted to the user who owns the resource. For example, administrators can change passwords for any user in their organisation.

### Super User Accounts

In addition to organisation administrators, Kerberos supports super user accounts that have access to all auth API paths across all organisations. These accounts are typically used for system administration and are configured separately from regular organisation administrators. Super user accounts bypass most authorization checks and are intended for use by the administration API (note: the admin functionality is being moved and is not covered in this documentation).

### Creating Administrator Accounts

When a new organisation is created, an administrator account is automatically generated with:
- Username: `admin-{organisation-name}`
- A randomly generated password (returned in the creation response)
- The `administrator` flag set to `true`

Additional users can be created within an organisation, but only existing administrators or super users can designate new users as administrators.
