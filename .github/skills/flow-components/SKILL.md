---
name: flow-components
description: 'Implement, extend, or debug FlowComponents in Kerberos. Covers the FlowComponent and Ordered interfaces, the fixed flow (Observability→Router→Custom→Forwarder), context guarantees, GetMeta conventions, and how to wire new components.'
argument-hint: 'Optional: the component to work on (observability, router, auth, oas, forwarder) or the task (add-component, debug-flow, inspect-meta)'
---

# Flow Components — Kerberos

## When to Use

- Adding a new component to the custom block (e.g., rate limiting, request rewriting)
- Modifying an existing FlowComponent's behaviour
- Debugging the request pipeline
- Understanding what context is available at a given point in the flow
- Adding metadata to the admin API flow endpoint

See [`docs/flow-components.md`](../../docs/flow-components.md) for a full reference.

---

## Interface Contracts

### `FlowComponent`

```go
// internal/composer/component.go
type FlowComponent interface {
    http.Handler              // ServeHTTP(http.ResponseWriter, *http.Request)
    Next(FlowComponent)       // Wire the next component; called once at startup
    GetMeta() []adminapi.FlowMeta
}
```

### `Ordered` (for custom-block components)

```go
// internal/composer/custom/component.go
type Ordered interface {
    FlowComponent
    Order() int  // Lower value = runs first. Minimum 1.
}
```

Implement `Ordered` when the component belongs in the custom block and needs a deterministic position relative to other custom-block components. The `order` value comes from the config section and should be exposed via the `ordered_schema.json` reference in the component's config schema.

---

## The Fixed flow

```
Observability → Router → Custom → Forwarder
```

Every request passes through this order. The Forwarder is always last and panics if `Next` is called on it.

---

## Request Context Guarantees

Custom-block components (and the Forwarder) can rely on the following being in the request context when their `ServeHTTP` is called:

| Key constant | Type | Set by | How to retrieve |
|---|---|---|---|
| `composer.BackendContextKey` (`"krb.backend"`) | `string` | Observability | `req.Context().Value(composer.BackendContextKey).(string)` |
| `composer.DebugContextKey` (`"krb.debug"`) | `debug.DebuggedCall` | Observability | `composer.DebugFromContext(req.Context())` |
| logger | `logr.Logger` | Observability | `logr.FromContext(req.Context())` |
| `composer.TargetContextKey` (`"krb.target"`) | `*config.RouterBackend` | Router | `req.Context().Value(composer.TargetContextKey).(*config.RouterBackend)` |

Always use `composer.DebugFromContext(ctx)` — it returns a no-op if no debug call is present, so no nil check is needed.

Do not assume any other custom-block component has run. If ordering matters, set `order` values explicitly in config.

---

## Implementing a Custom-Block Component

A custom-block component must implement `FlowComponent` and, if ordering is needed, `Ordered`.

### 1. Create the component struct and constructor

```go
// internal/mycomponent/component.go
package mycomponent

import (
    "net/http"

    "github.com/trebent/kerberos/internal/composer"
    "github.com/trebent/kerberos/internal/composer/custom"
    "github.com/trebent/kerberos/internal/config"
    adminapi "github.com/trebent/kerberos/internal/oapi/admin"
)

type (
    MyComponent interface {
        composer.FlowComponent
        custom.Ordered
    }
    Opts struct {
        Cfg *config.MyComponentConfig
    }
    myComponent struct {
        next composer.FlowComponent
        cfg  *config.MyComponentConfig
    }
)

var _ MyComponent = (*myComponent)(nil)

func NewComponent(opts *Opts) MyComponent {
    return &myComponent{cfg: opts.Cfg}
}
```

### 2. Implement `FlowComponent`

```go
func (c *myComponent) Next(next composer.FlowComponent) {
    c.next = next
}

func (c *myComponent) ServeHTTP(w http.ResponseWriter, req *http.Request) {
    // Read guaranteed context values:
    backend := req.Context().Value(composer.BackendContextKey).(string)
    debugCall := composer.DebugFromContext(req.Context())
    _ = backend
    _ = debugCall

    // ... do work ...

    // Forward to the next component (or short-circuit by writing a response and returning).
    c.next.ServeHTTP(w, req)
}

func (c *myComponent) GetMeta() []adminapi.FlowMeta {
    fmd := adminapi.FlowMeta_Data{}
    // Populate fmd using the appropriate From* method, e.g.:
    // fmd.FromNoFlowMetaData(adminapi.NoFlowMetaData{})

    return append([]adminapi.FlowMeta{
        {Name: "my-component", Data: fmd},
    }, c.next.GetMeta()...)
}
```

### 3. Implement `Ordered`

```go
func (c *myComponent) Order() int {
    return c.cfg.Order
}
```

### 4. Add a config type and schema

In `internal/config/types.go`:

```go
MyComponentConfig struct {
    Order int `json:"order"`
    // ... other fields
}
```

Add `*MyComponentConfig` to `RootConfig` with an appropriate JSON tag and add a schema file at `internal/config/schemas/mycomponent_schema.json`. Embed it in `internal/config/config.go` alongside the other schemas. Reference `ordered_schema.json` for the `order` field:

```json
{
  "$schema": "http://json-schema.org/draft-07/schema#",
  "$id": "http://trebent.com/kerberos/schemas/mycomponent_schema.json",
  "type": "object",
  "properties": {
    "order": { "$ref": "http://trebent.com/kerberos/schemas/ordered_schema.json" }
  },
  "additionalProperties": false
}
```

### 5. Wire in `main.go`

```go
if cfg.MyComponentEnabled() {
    myComp := mycomponent.NewComponent(&mycomponent.Opts{Cfg: cfg.MyComponentConfig})
    customFlowComponents = append(customFlowComponents, myComp)
}
```

The `custom.NewComponent(customFlowComponents...)` call sorts by `Order()` and flows them automatically.

---

## `GetMeta` Conventions

- Every non-terminal component should **prepend** its own `FlowMeta` and **append** `c.next.GetMeta()`.
- The Forwarder (terminal) returns a single entry and does not call `next`.
- The Custom container does not add its own entry — it delegates to its first sub-component.
- Use the `From*` methods on `adminapi.FlowMeta_Data` (generated from the admin OAS) to set the data union type.
- If a component has no meaningful metadata, use `fmd.FromNoFlowMetaData(adminapi.NoFlowMetaData{})`.

---

## `DebuggedCall` Usage

Record a transition whenever a component finishes processing:

```go
debugCall.AddTransition(
    "my-component",                    // component name (matches GetMeta Name)
    debug.CallDirectionInbound,        // or CallDirectionOutbound
    debugStart,                        // time.Time at start of processing
    time.Now(),                        // time.Time at end of processing
    debug.CallResultSuccess,           // or CallResultFailure
    "",                                // error message on failure, else ""
)
```

Call this before forwarding to `next` (inbound) and optionally after it returns (outbound) if the component does post-processing.

---

## Useful File Locations

| File | Purpose |
|---|---|
| `internal/composer/component.go` | `FlowComponent` interface and `Dummy` |
| `internal/composer/composer.go` | `Composer` and flow wiring |
| `internal/composer/context.go` | Context key constants |
| `internal/composer/custom/component.go` | Custom container and `Ordered` interface |
| `internal/composer/debug/` | `DebuggedCall` types |
| `internal/auth/component.go` | Example: Authorizer (custom-block, Ordered) |
| `internal/oas/component.go` | Example: OAS Validator (custom-block, Ordered) |
| `internal/composer/forwarder/component.go` | Terminal component reference |
| `main.go` | Component wiring |
