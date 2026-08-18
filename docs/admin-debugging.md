# Admin Debugging

Kerberos provides a debugging feature that records the full flow of individual requests as they pass through the gateway. Debugging is scoped per backend and time-boxed to a configurable session duration, so it can be safely enabled in a running environment without lasting impact.

## How It Works

When a debug session is active for a backend:

1. The Observability flow component creates a `DebuggedCall` object and places it in the request context instead of the usual no-op.
2. Each flow component (Observability, Router, Auth, OAS Validator, Forwarder) records a _flow transition_ into the call as it starts and finishes processing.
3. After the response is sent, the call is finalised and persisted to the database.
4. The recorded calls can be retrieved via the admin API for inspection.

A rate limit of 100 calls per second applies across all active debug sessions to limit overhead.

---

## Permissions

All debug endpoints require the `debugger` permission. The super user account always has this permission. Regular admin users need the permission assigned explicitly.

---

## Debug Session Lifecycle

```
Start session → (session active) → calls are recorded → Stop / Delete / session expires
```

- **Start**: Opens a debug session for a backend, with a configurable expiry duration.
- **Extend**: Pushes the expiry further out. The total session lifetime cannot exceed one hour from start.
- **Stop**: Marks the session as stopped (no new calls are recorded) but keeps the session and its calls available for retrieval.
- **Delete**: Permanently removes the session and all associated call records.

Only one active session per backend is allowed at a time. Starting a session when one is already active returns `409 Conflict`.

---

## API Reference

All endpoints are authenticated. They require an active admin session (via the admin login flow) and the `debugger` permission.

### Debug Sessions

#### Start a debug session

```
POST /api/admin/debug/{backend}/sessions
```

Request body (optional):

```json
{
  "durationSeconds": 300
}
```

| Field | Description |
|---|---|
| `durationSeconds` | How long (in seconds) the session should remain active. Minimum `60`, maximum `3600`. Defaults to `300` (5 minutes). |

Returns `200` with the created `DebugSession` object, or `409` if an active session already exists.

#### List debug sessions

```
GET /api/admin/debug/{backend}/sessions
```

Returns all sessions (including expired and stopped ones) for the given backend.

#### Get a debug session

```
GET /api/admin/debug/{backend}/sessions/{sessionId}
```

Returns the `DebugSession` with the given ID.

#### Stop a debug session

```
POST /api/admin/debug/{backend}/sessions/{sessionId}
```

Marks the session as stopped. Recording halts immediately; previously captured calls remain available. Returns `204` on success.

#### Extend a debug session

```
PUT /api/admin/debug/{backend}/sessions/{sessionId}
```

Request body (required):

```json
{
  "additionalDurationSeconds": 300
}
```

Adds `additionalDurationSeconds` to the session's current expiry. The total session lifetime (measured from `startedAt`) cannot exceed one hour. Returns `200` with the updated session.

#### Delete a debug session

```
DELETE /api/admin/debug/{backend}/sessions/{sessionId}
```

Permanently deletes the session and all its call records. Returns `204`.

---

### Debug Session Calls

#### List calls for a session

```
GET /api/admin/debug/{backend}/sessions/{sessionId}/calls
```

Query parameters:

| Parameter | Description |
|---|---|
| `includeTransitions` | When `true`, each call includes its full list of flow transitions. Defaults to `false`. |

Returns an array of `DebugSessionCall` objects.

#### Get a specific call

```
GET /api/admin/debug/{backend}/sessions/{sessionId}/calls/{callId}
```

Returns a single `DebugSessionCall` including all its flow transitions.

---

## Data Model

### `DebugSession`

| Field | Description |
|---|---|
| `id` | Unique session identifier. |
| `backend` | The backend name this session is attached to. |
| `startedAt` | When the session was created. |
| `expiresAt` | When the session will stop recording new calls. |
| `stoppedAt` | When the session was manually stopped. `null` if still active. |

### `DebugSessionCall`

| Field | Description |
|---|---|
| `id` | Unique call identifier. |
| `method` | HTTP method of the request. |
| `url` | Request URL as seen by the gateway. |
| `statusCode` | HTTP status code returned to the client. |
| `startedAt` | When the gateway started processing the request. |
| `stoppedAt` | When the gateway finished sending the response. |
| `flowTransitions` | Ordered list of transitions recorded by each flow component. |

### `FlowTransition`

Each `FlowTransition` represents one component's processing window during the call:

| Field | Description |
|---|---|
| `component` | Name of the flow component (e.g. `observability`, `auth`, `forwarder`). |
| `direction` | `inbound` when the component is receiving the request; `outbound` when returning. |
| `startedAt` | When this transition began. |
| `stoppedAt` | When this transition ended. |
| `result.outcome` | `success` or `failure`. |
| `result.cause` | Non-empty only on `failure` — a short description of why the component rejected the request. |

---

## Typical Debugging Workflow

1. **Log in** to the admin API with an account that holds the `debugger` permission.
2. **Start a debug session** for the backend you want to inspect:
   ```
   POST /api/admin/debug/my-service/sessions
   {"durationSeconds": 120}
   ```
3. **Send one or more requests** through the gateway to the backend.
4. **List the captured calls**:
   ```
   GET /api/admin/debug/my-service/sessions/{sessionId}/calls?includeTransitions=true
   ```
5. **Inspect individual calls** to see which flow component rejected or delayed the request.
6. **Stop or delete the session** when done:
   ```
   POST /api/admin/debug/my-service/sessions/{sessionId}
   ```

---

## Notes

- Debugging adds a small per-request overhead (database write on call finalisation). Keep sessions short and targeted to production backends.
- The rate limiter silently drops recording (reverts to a no-op call) if the 100-calls/second threshold is exceeded; the request itself is still processed normally.
- Expired sessions are not automatically deleted. Use the delete endpoint to clean up old sessions and their call records.
