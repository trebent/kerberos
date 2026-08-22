# Routing

Kerberos handles incoming gateway requests in a fixed flow and only dispatches to a backend after context, routing, and optional policy checks are complete.

## Gateway request path format

Kerberos gateway routes are exposed under:

`/gw/backend/<backend-name>/<backend-path>`

`<backend-name>` identifies the configured target backend and `<backend-path>` is the forwarded path segment.

## High-level request lifecycle

1. **Observability entry**
   - Initializes request-level logging/debug context.
   - Starts tracing for the request.
   - Wraps request/response bodies for size/status measurement.

2. **Router resolution**
   - Reads `<backend-name>` from the gateway path.
   - Looks up the backend in configured router backends.
   - If no backend matches, returns `404` and stops processing.
   - Stores the resolved backend in request context and rewrites path by stripping the gateway prefix.

3. **Custom policy block (optional components)**
   - Executes configured components in configured order (for example auth and OAS validation).
   - Any component can reject the request (for example `401`, `403`, `400`) and stop forwarding.

4. **Forwarder dispatch**
   - Uses the resolved backend target (host/port/TLS/timeout settings) to send the upstream request.
   - Propagates request metadata, including trace context.

## High-level response lifecycle

1. Backend response is received by the forwarder.
2. Status, headers, and body are relayed back to the client.
3. Control returns up the flow so observability can finalize metrics, span state, and debug call data.

## Notes on behavior

- Flow order is fixed at runtime: Observability → Router → Custom block → Forwarder.
- Backend-specific behavior is driven by configuration and optional mapped components.
- Debug sessions (when active) capture transition data across all flow stages.

For internals of flow composition and context contracts, see [Flow Components](./dev/flow-components.md).
