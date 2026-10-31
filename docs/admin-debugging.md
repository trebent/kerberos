# Debug sessions

A debug session records how individual requests to one backend travel through the gateway: which steps of the pipeline handled each request, how long each step took, and where and why a request was rejected. Use it when a client reports an unexpected `401`, `403`, `400` or `500` and you want to see which part of Kerberos produced it, without turning on tracing for the whole gateway.

Debug sessions are managed through the Admin API on the admin port (30001 in the examples below).

## What a debug session records

While a session is active for a backend, each recorded request to `/gw/backend/<backend>/...` becomes a **call** with:

| Field | Description |
| --- | --- |
| `id` | Call ID, unique across all sessions. |
| `url` | The request path as the client sent it, including the `/gw/backend/<backend>` prefix, for example `/gw/backend/echo/hello`. The query string and host are not included. |
| `method` | HTTP method. |
| `statusCode` | Status code returned to the client. |
| `startedAt`, `stoppedAt` | When Kerberos received the request and when it finished handling it. |
| `flowTransitions` | One entry per pipeline step that handled the request, in chronological order (see below). |

Each **flow transition** has:

| Field | Description |
| --- | --- |
| `component` | The pipeline step, see the table below. |
| `direction` | `inbound` (handling the request on its way to the backend) or `outbound` (handling the response on its way back to the client). |
| `startedAt`, `stoppedAt` | Time spent in that step. |
| `result.outcome` | `success` or `failure`. |
| `result.cause` | For a failure, the reason, for example `Unauthorized` or the request validation error. Empty string for successes. |

The component names you will see, in pipeline order (see [Routing](./routing.md) for the full pipeline):

| Component | Directions | When it appears |
| --- | --- | --- |
| `obs` | inbound, outbound | Always, unless observability is disabled (`observability.enabled: false`). Calls are still recorded when observability is disabled, but without `obs` transitions. |
| `router` | inbound | Always. Fails when the backend is unknown (`no backend found`) or the request `Origin` is not allowed. A successful CORS preflight (`OPTIONS`) ends here. |
| `authorizer` | inbound | Only if the backend has an `auth` mapping. Fails with `Unauthorized` or `Forbidden`. See [Authentication](./authentication.md). |
| `oas-validator` | inbound | Only if the backend has an `oas` mapping. Fails with the validation error. See [Request validation](./request-validation.md). |
| `forwarder` | inbound, outbound | When the request reaches the backend. `inbound` covers sending the request and waiting for the backend's response headers; `outbound` covers copying the response back to the client. A failure here means the backend could not be reached or the response could not be relayed. |

`authorizer` and `oas-validator` run in the order set by their `order` setting. A step that rejects a request is the last `inbound` transition of the call; the `obs` outbound transition still follows it.

## Permissions

All debug endpoints require an admin session (`401` without one) with the `debugger` permission (`403` without it). The superuser has every permission, including `debugger`. To give an admin user access, put them in an admin group that has the `debugger` permission. See [Admin API](./admin-api.md) for logging in, admin groups and the permission list.

## Endpoints

All paths are under `/api/admin/debug/{backend}/sessions` on the admin port. `{backend}` is the backend name from the router configuration.

| Operation | Method and path | Body / parameters | Success |
| --- | --- | --- | --- |
| Start a session | `POST /api/admin/debug/{backend}/sessions` | Optional `{"durationSeconds": 300}`, 60–3600, default 300 | `200` with the session |
| List sessions | `GET /api/admin/debug/{backend}/sessions` | | `200` with all sessions for the backend, active or not |
| Get a session | `GET /api/admin/debug/{backend}/sessions/{sessionId}` | | `200` with the session |
| Extend a session | `PUT /api/admin/debug/{backend}/sessions/{sessionId}` | `{"additionalDurationSeconds": 600}`, at least 1 | `200` with the updated session |
| Stop a session | `POST /api/admin/debug/{backend}/sessions/{sessionId}` | | `204` |
| Delete a session | `DELETE /api/admin/debug/{backend}/sessions/{sessionId}` | | `204` |
| List calls | `GET /api/admin/debug/{backend}/sessions/{sessionId}/calls?includeTransitions=true` | `includeTransitions` is required | `200` with calls, newest first |
| Get a call | `GET /api/admin/debug/{backend}/sessions/{sessionId}/calls/{callId}` | | `200` with the call and its transitions |

A session object looks like this:

```json
{
  "id": 4,
  "backend": "echo",
  "startedAt": "2026-10-29T10:15:02Z",
  "expiresAt": "2026-10-29T10:20:02.418230911Z",
  "stoppedAt": null
}
```

`stoppedAt` is set once the session has been stopped. Errors use the usual `{"errors": ["..."]}` format. The full contract is in [`openapi/admin.yaml`](../openapi/admin.yaml), also available as a [rendered API reference](https://trebent.github.io/kerberos/api/admin).

### Lifecycle

- **Start.** One session can be active per backend. Starting another while one is active returns `409 Conflict`. A session is active until it expires or is stopped. Use the backend name exactly as it appears in the router configuration.
- **Recording.** Calls are recorded until `expiresAt` or until the session is stopped.
- **Extend.** Adds `additionalDurationSeconds` to the current `expiresAt`. If the result would be more than one hour from now, `expiresAt` is set to one hour after `startedAt` instead. Extending also starts recording again for a session that was stopped or has expired, as long as the new `expiresAt` is in the future.
- **Stop.** Ends recording immediately and sets `stoppedAt`. Captured calls are kept.
- **Delete.** Removes the session and all of its captured calls and transitions.

Sessions and calls are stored in the gateway database (see [Persistence](./persistence.md)) and are kept until you delete them.

## Walkthrough

This example assumes a backend called `echo`, the gateway on port 30000 and the admin API on port 30001.

1. Log in as the superuser (or as an admin user with `debugger`) and keep the cookies:

   ```bash
   curl -s -c cookies.txt -X POST http://localhost:30001/api/admin/superuser/login \
     -H 'Content-Type: application/json' \
     -d '{"clientId": "admin", "clientSecret": "secret"}'
   ```

   Admin cookies are marked `Secure`, which curl accepts for `localhost`. For other hosts use HTTPS; see [Sessions](./sessions.md).

2. Start a 10-minute session for `echo`:

   ```bash
   curl -s -b cookies.txt -X POST http://localhost:30001/api/admin/debug/echo/sessions \
     -H 'Content-Type: application/json' \
     -d '{"durationSeconds": 600}'
   ```

   ```json
   {
     "id": 4,
     "backend": "echo",
     "startedAt": "2026-10-29T10:15:02Z",
     "expiresAt": "2026-10-29T10:25:02.418230911Z",
     "stoppedAt": null
   }
   ```

3. Reproduce the problem through the gateway:

   ```bash
   curl -s http://localhost:30000/gw/backend/echo/hello
   ```

4. List the captured calls with their transitions:

   ```bash
   curl -s -b cookies.txt \
     'http://localhost:30001/api/admin/debug/echo/sessions/4/calls?includeTransitions=true'
   ```

   For a backend with an `auth` mapping, a request without a valid login session looks like this:

   ```json
   [
     {
       "id": 31,
       "url": "/gw/backend/echo/hello",
       "method": "GET",
       "statusCode": 401,
       "startedAt": "2026-10-29T10:16:40.112904Z",
       "stoppedAt": "2026-10-29T10:16:40.114387Z",
       "flowTransitions": [
         {
           "component": "obs",
           "direction": "inbound",
           "startedAt": "2026-10-29T10:16:40.112911Z",
           "stoppedAt": "2026-10-29T10:16:40.112958Z",
           "result": { "outcome": "success", "cause": "" }
         },
         {
           "component": "router",
           "direction": "inbound",
           "startedAt": "2026-10-29T10:16:40.112963Z",
           "stoppedAt": "2026-10-29T10:16:40.112989Z",
           "result": { "outcome": "success", "cause": "" }
         },
         {
           "component": "authorizer",
           "direction": "inbound",
           "startedAt": "2026-10-29T10:16:40.112994Z",
           "stoppedAt": "2026-10-29T10:16:40.113410Z",
           "result": { "outcome": "failure", "cause": "Unauthorized" }
         },
         {
           "component": "obs",
           "direction": "outbound",
           "startedAt": "2026-10-29T10:16:40.113452Z",
           "stoppedAt": "2026-10-29T10:16:40.113501Z",
           "result": { "outcome": "success", "cause": "" }
         }
       ]
     }
   ]
   ```

   The `authorizer` failure shows that the request stopped there, before validation and the backend. A successful call would instead show `authorizer` and `oas-validator` (if configured) succeeding, followed by `forwarder` inbound and outbound.

   With `includeTransitions=false` the same list is returned with `"flowTransitions": null`, which is faster when a session has many calls.

5. Look at a single call in full:

   ```bash
   curl -s -b cookies.txt http://localhost:30001/api/admin/debug/echo/sessions/4/calls/31
   ```

6. Extend the session if you need more time, then stop it when you are done:

   ```bash
   curl -s -b cookies.txt -X PUT http://localhost:30001/api/admin/debug/echo/sessions/4 \
     -H 'Content-Type: application/json' \
     -d '{"additionalDurationSeconds": 600}'

   curl -s -b cookies.txt -X POST http://localhost:30001/api/admin/debug/echo/sessions/4
   ```

7. Delete it once you no longer need the captured calls:

   ```bash
   curl -s -b cookies.txt -X DELETE http://localhost:30001/api/admin/debug/echo/sessions/4
   ```

## Things to know

- **Built for a handful of requests.** Debug sessions work best when you reproduce a problem with a few requests. Kerberos records about one request per second, with bursts of up to 100, across the whole gateway, and every request is still handled normally.
- **One instance.** A debug session records the requests handled by the Kerberos instance that started it. After a restart, start or extend a session to record again. See [Deployment](./deployment.md).
- **One active session per backend.** Extend the active session to record for longer.
- **Stop sessions when you are done**, and delete them once you no longer need the calls.
- **Backend paths only.** Requests to `/gw/backend/<name>/...` are recorded.

## Privacy

A debug session stores only basic request details: method, path, status code, timings and the result of each step. It does **not** store request or response headers, cookies, bodies, query strings, client addresses or the identity of the caller.

Some stored values can still contain sensitive data:

- The `url` path, if your API puts identifiers or other personal data in paths.
- Failure causes. Request validation errors can quote parts of the request, and backend connection errors include the backend's internal address.

Debug data stays in the database until the session is deleted, and anyone with the `debugger` permission can read every session. Delete sessions when you have finished investigating.

## Debug sessions compared with tracing

[Observability](./observability.md) covers tracing, metrics and logs. Tracing is meant to be always on, covers all backends, and sends its data to an OpenTelemetry collector where it joins up with your backend's own traces. Debug sessions are short, cover one backend, need nothing else to be installed, and show the result and cause for each gateway step, including requests that were rejected before reaching the backend. Use tracing for production monitoring and latency analysis across services; use a debug session to find out why particular requests are failing at the gateway.

## Related docs

- [Admin API](./admin-api.md): logging in, admin users and groups, permissions
- [Routing](./routing.md): the request pipeline and `/gw/backend/...` paths
- [Authentication](./authentication.md) and [Request validation](./request-validation.md): the steps that most often reject requests
- [Observability](./observability.md): tracing, metrics and logs
- [`openapi/admin.yaml`](../openapi/admin.yaml): full API contract, also [rendered](https://trebent.github.io/kerberos/api/admin)
