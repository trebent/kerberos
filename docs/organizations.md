# Organizations

Organizations (or "organisations" in the codebase) are the top-level entity in Kerberos' multi-tenant authentication system. They provide isolation between different groups of users and their resources.

## Organization Structure

Each organization contains:

- **Users**: Individual accounts that can authenticate and access resources
- **Groups**: Named collections used for authorization
- **Group Bindings**: Associations between users and groups
- **Sessions**: Active authentication sessions for users

All users, groups, and sessions are scoped to a single organisation, ensuring complete isolation between tenants.

## Working with Organizations

### Creating an Organization

Organizations are created through the basic authentication API:

```
POST /auth/basic/organisations
{
  "name": "my-organization"
}
```

When an organization is created:

1. The organization record is created in the database with a unique ID
2. An administrator account is automatically created with:
   - Username: `admin-{organisation-name}`
   - A randomly generated password (returned in the response)
   - Full administrative privileges within the organization

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

### Listing Organizations

Super user accounts can list all organizations:

```
GET /auth/basic/organisations
```

Regular users (including organization administrators) cannot list organizations - this operation is restricted to super users only.

### Managing an Organization

Organization administrators can:

- View organization details: `GET /auth/basic/organisations/{orgID}`
- Update organization name: `PUT /auth/basic/organisations/{orgID}`
- Delete organization: `DELETE /auth/basic/organisations/{orgID}`

Deleting an organization will cascade delete all associated users, groups, group bindings, and sessions due to database foreign key constraints.

## Organization Administrator Accounts

Organization administrator accounts have special privileges that allow them to manage their organization.

### Administrator Capabilities

An organization administrator can:

1. **Manage Users**
   - Create new users: `POST /auth/basic/organisations/{orgID}/users`
   - List all users: `GET /auth/basic/organisations/{orgID}/users`
   - View any user: `GET /auth/basic/organisations/{orgID}/users/{userID}`
   - Update any user: `PUT /auth/basic/organisations/{orgID}/users/{userID}`
   - Delete any user: `DELETE /auth/basic/organisations/{orgID}/users/{userID}`
   - Change any user's password: `POST /auth/basic/organisations/{orgID}/users/{userID}/password`

2. **Manage Groups**
   - Create groups: `POST /auth/basic/organisations/{orgID}/groups`
   - List all groups: `GET /auth/basic/organisations/{orgID}/groups`
   - View group details: `GET /auth/basic/organisations/{orgID}/groups/{groupID}`
   - Update group names: `PUT /auth/basic/organisations/{orgID}/groups/{groupID}`
   - Delete groups: `DELETE /auth/basic/organisations/{orgID}/groups/{groupID}`

3. **Manage Group Memberships**
   - View user's groups: `GET /auth/basic/organisations/{orgID}/users/{userID}/groups`
   - Update user's group memberships: `PUT /auth/basic/organisations/{orgID}/users/{userID}/groups`

4. **Organization Management**
   - View organization details
   - Update organization name
   - Delete organization (and all associated data)

### Regular User Limitations

Regular users (non-administrators) have much more limited access:

- They can only view their own user details (`GET /auth/basic/organisations/{orgID}/users/{userID}` where userID matches their own)
- They cannot create, update, or delete users
- They cannot manage groups or group memberships
- They cannot perform any organization-level operations

### User Account Structure

Each user account in the system has the following properties:

- **ID**: Unique identifier
- **Name**: Username used for login
- **Organisation ID**: The organization this user belongs to
- **Salt & Hashed Password**: Secure password storage
- **Administrator**: Boolean flag indicating if the user has administrator privileges
- **Super User**: Boolean flag for system-level super user accounts (used by admin API)

The `administrator` flag is what grants the elevated privileges within an organization. This flag can only be set by:
- Super user accounts when creating a user
- The initial administrator account created with the organization

## Best Practices

### Security

1. **Change default passwords**: Immediately change the administrator password after organization creation
2. **Limit administrator accounts**: Only grant administrator privileges to users who need them
3. **Use groups for authorization**: Define groups that map to your authorization requirements
4. **Regular auditing**: Periodically review user and group memberships

### Organization Design

1. **One organization per tenant**: Each customer or isolated environment should have its own organization
2. **Meaningful group names**: Use group names that clearly indicate their purpose
3. **Document group purposes**: Keep external documentation of what each group is authorized to do
4. **Plan for growth**: Design your group structure to accommodate future authorization requirements

### User Management

1. **Consistent naming**: Establish a naming convention for users (e.g., email addresses)
2. **Onboarding process**: Have a standard process for creating users and assigning groups
3. **Offboarding process**: Remove users or revoke their group memberships when they should no longer have access
4. **Password policies**: Enforce strong passwords (note: password strength is not currently enforced by the API)
