# Organisations

Organisations are the top-level entity in Kerberos' multi-tenant authentication system. They provide isolation between different groups of users and their resources.

## Organisation Structure

Each organisation contains:

- **Users**: Individual accounts that can authenticate and access resources
- **Groups**: Named collections used for authorization
- **Group Bindings**: Associations between users and groups
- **Sessions**: Active authentication sessions for users

All users, groups, and sessions are scoped to a single organisation, ensuring complete isolation between tenants.

## Working with Organisations

### Creating an Organisation

Organisations are created through the basic authentication API:

```
POST /api/auth/basic/organisations
{
  "name": "my-organization"
}
```

When an organisation is created:

1. The organisation record is created in the database with a unique ID
2. An administrator account is automatically created with:
   - Username: `admin-{organisation-name}`
   - A randomly generated password (returned in the response)
   - Full administrative privileges within the organisation

The creation response includes:

```json
{
  "id": 1,
  "name": "my-organization",
  "adminUserId": 2,
  "adminUsername": "admin-my-organization",
  "adminPassword": "generated-password"
}
```

**Important**: Save the administrator password from the creation response, as it cannot be retrieved later. The administrator can change their password after logging in.

### Listing Organisations

Super user accounts can list all organisations:

```
GET /api/auth/basic/organisations
```

Regular users (including organisation administrators) cannot list organisations - this operation is restricted to super users only.

### Managing an Organisation

Organisation administrators can:

- View organisation details: `GET /api/auth/basic/organisations/{orgID}`
- Update organisation name: `PUT /api/auth/basic/organisations/{orgID}`
- Delete organisation: `DELETE /api/auth/basic/organisations/{orgID}`

Deleting an organisation will cascade delete all associated users, groups, group bindings, and sessions due to database foreign key constraints.

## Organisation Administrator Accounts

Organisation administrator accounts have special privileges that allow them to manage their organisation.

### Administrator Capabilities

An organisation administrator can:

1. **Manage Users**
   - Create new users: `POST /api/auth/basic/organisations/{orgID}/users`
   - List all users: `GET /api/auth/basic/organisations/{orgID}/users`
   - View any user: `GET /api/auth/basic/organisations/{orgID}/users/{userID}`
   - Update any user: `PUT /api/auth/basic/organisations/{orgID}/users/{userID}`
   - Delete any user: `DELETE /api/auth/basic/organisations/{orgID}/users/{userID}`
   - Change any user's password: `PUT /api/auth/basic/organisations/{orgID}/users/{userID}/password`

2. **Manage Groups**
   - Create groups: `POST /api/auth/basic/organisations/{orgID}/groups`
   - List all groups: `GET /api/auth/basic/organisations/{orgID}/groups`
   - View group details: `GET /api/auth/basic/organisations/{orgID}/groups/{groupID}`
   - Update group names: `PUT /api/auth/basic/organisations/{orgID}/groups/{groupID}`
   - Delete groups: `DELETE /api/auth/basic/organisations/{orgID}/groups/{groupID}`

3. **Manage Group Memberships**
   - View user's groups: `GET /api/auth/basic/organisations/{orgID}/users/{userID}/groups`
   - Update user's group memberships: `PUT /api/auth/basic/organisations/{orgID}/users/{userID}/groups`

4. **Organisation Management**
   - View organisation details
   - Update organisation name
   - Delete organisation (and all associated data)

### Regular User Capabilities

Regular users (non-administrators) have limited but essential self-management capabilities:

- They can view their own user details (`GET /api/auth/basic/organisations/{orgID}/users/{userID}` where userID matches their own)
- They can update their own user information (`PUT /api/auth/basic/organisations/{orgID}/users/{userID}` for their own userID)
- They can change their own password (`PUT /api/auth/basic/organisations/{orgID}/users/{userID}/password` for their own userID)
- They cannot create, update, or delete other users
- They cannot manage groups or group memberships
- They cannot perform any organisation-level operations

### User Account Structure

Each user account in the system has the following properties:

- **ID**: Unique identifier
- **Name**: Username used for login
- **Organisation ID**: The organisation this user belongs to
- **Salt & Hashed Password**: Secure password storage
- **Administrator**: Boolean flag indicating if the user has administrator privileges

The `administrator` flag is what grants the elevated privileges within an organisation. In the current implementation, this flag is set automatically only for the initial administrator account created with the organisation. Additional administrator accounts cannot be created or modified through the API.

## Best Practices

### Security

1. **Change default passwords**: Immediately change the administrator password after organisation creation
2. **Limit administrator accounts**: Only grant administrator privileges to users who need them
3. **Use groups for authorization**: Define groups that map to your authorization requirements
4. **Regular auditing**: Periodically review user and group memberships

### Organisation Design

1. **One organisation per tenant**: Each customer or isolated environment should have its own organisation
2. **Meaningful group names**: Use group names that clearly indicate their purpose
3. **Document group purposes**: Keep external documentation of what each group is authorized to do
4. **Plan for growth**: Design your group structure to accommodate future authorization requirements

### User Management

1. **Consistent naming**: Establish a naming convention for users (e.g., email addresses)
2. **Onboarding process**: Have a standard process for creating users and assigning groups
3. **Offboarding process**: Remove users or revoke their group memberships when they should no longer have access
4. **Password policies**: Enforce strong passwords (note: password strength is not currently enforced by the API)
