# Flow Components

Every incoming request travels through a **linked list of flow components** called the _flow_. Each component implements the `FlowComponent` interface, performs its work, and then delegates to the next component in the flow.

---

## The `FlowComponent` Interface

```go
type FlowComponent interface {
    http.Handler            // ServeHTTP(http.ResponseWriter, *http.Request)
    Next(FlowComponent)     // Set the next component in the flow
    GetMeta() []adminapi.FlowMeta
}
```

- **`ServeHTTP`** — processes the request/response. A component may short-circuit (write a response and return without calling `next`) or forward to the next component by calling `next.ServeHTTP(w, req)`.
- **`Next`** — called once at startup to wire the linked list. Components store the reference and use it inside `ServeHTTP`.
- **`GetMeta`** — returns metadata about this component and all components that follow it. Each component prepends its own `FlowMeta` entry and appends `next.GetMeta()` so the admin API can inspect the entire flow.

---

## The Fixed flow

At startup the `composer` package wires four components into an immutable flow:

```txt
Observability → Router → Custom → Forwarder
```

Every request passes through them in this order.

### Observability

**Package:** `internal/composer/observability`

The first component in the flow. It:

- Extracts any incoming OTEL trace context from request headers and starts a new span.
- Extracts the backend name from the URL path and stores it under `krb.backend` in the request context.
- Initialises a `DebuggedCall` (stored under `krb.debug`) and a structured logger (stored via `logr.NewContext`).
- Wraps the request body if one is present to capture its size.
- Wraps the `http.ResponseWriter` to capture status code and response size.
- Records request and response metrics (count, size, duration) after the downstream flow returns.

When observability is disabled via config, a lightweight dummy component is used instead. It still performs the essential steps (extracting backend name, starting a debug call, setting the logger and response wrapper) so that downstream components always see a fully populated context.

### Router

**Package:** `internal/composer/router`

Matches the backend name already stored in `krb.backend` against the configured backends, then:

- Stores the matched `*config.RouterBackend` under `krb.target` in the request context.
- Updates the response wrapper's internal request context so higher-level middleware can read it.
- Strips the `/gw/backend/<backend-name>` prefix from `req.URL.Path` before forwarding.

If no backend matches, the router writes a `404` response and the flow stops.

### Custom

**Package:** `internal/composer/custom`

A container that holds **zero or more** optional sub-components. It delegates to its internal linked list of sub-components, then passes control to the Forwarder. See [Custom-Block Components](#custom-block-components) below for details.

### Forwarder

**Package:** `internal/composer/forwarder`

The terminal component — it does not accept a `Next` call (panics if one is attempted). It:

- Reads `krb.target` from the context to determine the backend.
- Forwards the request using a pre-configured `*http.Client` for that backend.
- Copies the backend response headers and body back to the original `http.ResponseWriter`.

---

## Custom-Block Components

The Custom component acts as a container. Its sub-components are sorted by their `Order()` value at startup, then arranged into an ordered sub-flow. The last sub-component's `Next` is wired to the Forwarder.

When there are zero sub-components, the Custom component forwards directly to the Forwarder.

### The `Ordered` Interface

```go
type Ordered interface {
    FlowComponent
    Order() int // lower value = runs first; minimum 1
}
```

Sub-components that implement `Ordered` are sorted by their `Order()` value. Components that do _not_ implement `Ordered` are appended after all ordered ones.

### Currently Provided Custom-Block Components

| Component | Package | Config section | Default order |
|---|---|---|---|
| **Authorizer** | `internal/auth` | `auth` | configurable via `auth.order` |
| **OAS Validator** | `internal/oas` | `oas` | configurable via `oas.order` |

Both are optional and only included when their respective config sections are present.

**Authorizer** — Checks that the request is authenticated (session header) and authorised (group membership) against the mapping defined in the `auth` config. Calls `next` on success or exempt paths; writes `401`/`403` on failure.

**OAS Validator** — Validates the incoming request (path, method, and optionally body) against the OpenAPI specification mapped to the current backend. Calls `next` if validation passes; writes `400` on failure. Backends without a spec mapping are passed through unchanged.

---

## Request Context Guarantees

Components in the custom block (and the Forwarder) can rely on the following values being present in the request context before they run:

| Context key | Type | Set by | Value |
|---|---|---|---|
| `krb.backend` (`composer.BackendContextKey`) | `string` | Observability | The backend name extracted from the request URL |
| `krb.debug` (`composer.DebugContextKey`) | `debug.DebuggedCall` | Observability | A debug call object for recording transitions; use `composer.DebugFromContext(ctx)` to retrieve it |
| logger | `logr.Logger` | Observability | Structured request logger; use `logr.FromContext(ctx)` to retrieve it |
| `krb.target` (`composer.TargetContextKey`) | `*config.RouterBackend` | Router | The matched backend configuration including host, port, and TLS settings |

> **Note:** Custom-block components run _after_ the Router, so both `krb.backend` and `krb.target` are always available. They should not assume any particular ordering relative to _other_ custom-block components — if a component depends on another (e.g., auth before OAS validation), configure their `order` values explicitly.

Use `composer.DebugFromContext(ctx)` to safely get the debug call — it returns a no-op implementation if none is found, so components never need to nil-check it.

---

## `GetMeta` and the Admin API

Each component prepends a `FlowMeta` entry and appends the downstream flow's metadata:

```go
return append([]adminapi.FlowMeta{{Name: "my-component", Data: fmd}}, c.next.GetMeta()...)
```

The admin API endpoint `GET /api/admin/flow` calls `GetMeta` on the head of the flow (Observability) and returns the full slice. This allows operators to inspect the active flow and its configuration at runtime.

The Forwarder, as the terminal component, returns a single entry with no downstream call. The Custom container does not add its own entry — it delegates directly to its first sub-component's `GetMeta`.

---

## Dummy Component

`composer.Dummy` is a pass-through component used internally when a component is disabled (e.g., Observability with `enabled: false`). It forwards `ServeHTTP` and `GetMeta` to `next` without modification. An optional `CustomHandler` function can be set to intercept the call.
