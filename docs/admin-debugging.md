# Admin Debugging

Kerberos supports request-flow debugging through the Admin API. Debugging records flow transitions for requests on selected backends during active debug sessions.

## How it works

When a debug session is active for a backend:

1. A debug call context is initialized for matching requests.
2. Flow components record transition events while processing.
3. Finalized call records are persisted.
4. Records are available through admin debug endpoints.

A global rate limit protects runtime overhead during active debugging.

## Access and permissions

Debug endpoints require an authenticated admin session with debugger access. Superuser sessions also have access.

## Session lifecycle

Typical lifecycle:

`start -> active capture -> stop/delete/expiry`

Core endpoint families:

- Session management: `/api/admin/debug/{backend}/sessions` and `/api/admin/debug/{backend}/sessions/{sessionId}`
- Captured calls: `/api/admin/debug/{backend}/sessions/{sessionId}/calls` and `/api/admin/debug/{backend}/sessions/{sessionId}/calls/{callId}`

## Typical workflow

1. Authenticate to Admin API.
2. Start a debug session for a backend.
3. Send requests through the gateway.
4. Retrieve captured calls.
5. Stop or delete the session.

For exact request/response bodies, query parameters, and status codes, use [`openapi/admin.yaml`](../openapi/admin.yaml).
