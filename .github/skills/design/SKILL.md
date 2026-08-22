---
name: design
description: 'Design guidelines for Kerberos Go code. Covers DRY/KISS, Opts structs, generics, interface patterns, error handling, constructors, and context propagation. Use this skill before preparing any implementation.'
argument-hint: 'Optional: the rule area to focus on (dry, kiss, opts, generics, interfaces, errors, constructors, context)'
---

# Design Guidelines — Kerberos

## When to Use

Use this skill **before preparing any implementation** in this repository. It defines
the non-negotiable design rules all Kerberos code must follow. Read it alongside the
relevant feature skill (`configuration`, `flow-components`, `oas`, `validation`).

---

## System Understanding

Before writing any code, read `docs/README.md` to orient yourself. Then follow the
links to the topic docs that are relevant to your change:

| Doc | Covers |
|---|---|
| [`docs/configuration.md`](../../../docs/configuration.md) | Config loading pipeline, reference syntax, all config sections |
| [`docs/dev/flow-components.md`](../../../docs/dev/flow-components.md) | FlowComponent interface and the request pipeline |
| [`docs/authentication.md`](../../../docs/authentication.md) | Auth system and the authorizer middleware |
| [`docs/routing.md`](../../../docs/routing.md) | HTTP handler call order |
| [`docs/observability.md`](../../../docs/observability.md) | Metrics and tracing |
| [`docs/dev/middleware.md`](../../../docs/dev/middleware.md) | oapi-codegen middleware ordering rules |
| [`docs/organizations.md`](../../../docs/organizations.md) | Multi-tenant organisation model |

Understanding the system layout prevents mis-placed code and wrong abstractions.

---

## Design Rules

### 1. DRY — Don't Repeat Yourself

Extract shared logic rather than duplicating it. If the same expression, block, or
concept appears more than once, give it a name and a home.

```go
// BAD — duplicated nil check
if cfg.AuthConfig == nil {
    return errors.New("auth not configured")
}
// ... elsewhere ...
if cfg.AuthConfig == nil {
    return errors.New("auth not configured")
}

// GOOD — single helper
func (rc *RootConfig) AuthEnabled() bool {
    return rc.AuthConfig != nil
}
```

### 2. KISS — Keep It Simple

Prefer the simplest solution that correctly satisfies the requirement. Do not add
abstractions, layers, or configuration switches for hypothetical future needs.

- Solve the problem at hand; don't pre-optimise.
- If two approaches are equally correct, choose the one a new reader understands first.
- Delete code that is no longer used.

### 3. Opts Structs for High Parameter Count

When a function or constructor accepts more than approximately three parameters, group
them in an `Opts` struct passed by pointer. This keeps call sites readable and makes
adding new parameters backwards-compatible.

```go
// BAD
func NewComponent(cfg *config.MyConfig, logger *slog.Logger, timeout time.Duration, enabled bool) Component

// GOOD
type Opts struct {
    Cfg     *config.MyConfig
    Logger  *slog.Logger
    Timeout time.Duration
    Enabled bool
}

func New(opts *Opts) Component
```

Rule of thumb: if you find yourself writing a function signature that wraps to a second
line, introduce an `Opts` struct.

### 4. Generics — Use Where Clearly Beneficial, Not Preemptively

Genericise code when the same algorithm or data structure is genuinely needed for
multiple concrete types with no meaningful behavioural difference between them.

```go
// GOOD — generic map helper used across the codebase
func MapSlice[T, U any](s []T, f func(T) U) []U {
    out := make([]U, len(s))
    for i, v := range s {
        out[i] = f(v)
    }
    return out
}
```

Do **not** reach for generics just because something _could_ be parameterised. If
there is only one concrete use today, write it concretely.

### 5. Compile-Time Interface Assertion

Verify that a struct implements its intended interface at compile time. Place the
assertion near the type declaration.

```go
var _ MyInterface = (*impl)(nil)
```

This turns a missing method into a compile error rather than a runtime panic.

### 6. Interfaces for Testability

Define interfaces at the **consumer** side. Pass dependencies as interfaces so tests
can substitute fakes or mocks without relying on the real implementation.

```go
// In the consumer package
type UserStore interface {
    GetUser(ctx context.Context, id string) (*User, error)
}

type handler struct {
    store UserStore
}
```

Avoid exposing concrete types as function parameters when the caller only needs a
subset of the type's behaviour.

### 7. Standard Go Interfaces

Implement stdlib interfaces where applicable. Do **not** invent custom equivalents.

| Use case | Use this |
|---|---|
| Byte-stream reading | `io.Reader` |
| Byte-stream writing | `io.Writer` |
| Close/cleanup | `io.Closer` |
| HTTP handling | `http.Handler` |
| Human-readable string | `fmt.Stringer` |
| Structured logging sink | `io.Writer` (pass to `slog.New(slog.NewJSONHandler(w, nil))`) |

Composing stdlib interfaces (`io.ReadCloser`, `io.ReadWriter`) is preferred over
defining new combined interfaces unless the combination would be surprising.

### 8. Error Wrapping

Always wrap errors with context using `fmt.Errorf` and the `%w` verb. This keeps
errors unwrappable for `errors.Is` / `errors.As` and preserves the call chain for
debugging.

```go
// BAD
return errors.New("failed to load user: " + err.Error())

// GOOD
return fmt.Errorf("load user %q: %w", id, err)
```

Context messages should be lowercase, concise noun-phrases that read naturally when
chained: `"parse config: open file: no such file or directory"`.

### 9. Constructor Pattern

Always expose `New(*Opts) Interface` as the public constructor. Keep struct fields
unexported. Callers must not construct types directly.

```go
type Opts struct {
    Cfg *config.MyConfig
}

type impl struct {
    cfg *config.MyConfig
}

var _ MyInterface = (*impl)(nil)

func New(opts *Opts) MyInterface {
    return &impl{cfg: opts.Cfg}
}
```

This makes the zero value non-functional by design, preventing accidental use of
uninitialised types.

### 10. No Naked Returns

Never rely on implicit `return` with named return values. Always write explicit
`return` statements, even in short functions. Named return values are fine for
documentation purposes (e.g. `(n int, err error)`) but must not be used as an
implicit return mechanism.

```go
// BAD
func divide(a, b float64) (result float64, err error) {
    if b == 0 {
        err = errors.New("division by zero")
        return
    }
    result = a / b
    return
}

// GOOD
func divide(a, b float64) (float64, error) {
    if b == 0 {
        return 0, errors.New("division by zero")
    }
    return a / b, nil
}
```

### 11. Context Propagation

Accept `context.Context` as the **first parameter** of any function that performs I/O,
makes network calls, or may block. Never store a context inside a struct field.

```go
// BAD — context stored in struct
type client struct {
    ctx context.Context
}

// GOOD — context threaded through calls
func (c *client) Fetch(ctx context.Context, id string) (*Resource, error) {
    req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.url+"/"+id, nil)
    if err != nil {
        return nil, fmt.Errorf("build request: %w", err)
    }
    // ...
}
```

Pass the context through — do not use `context.Background()` inside a function that
already received a context from its caller.

---

## Quick Reference Checklist

Before opening a PR, verify your change satisfies every applicable rule:

- [ ] No logic duplicated — shared code extracted
- [ ] Simplest correct solution chosen — no speculative abstractions
- [ ] Functions with >3 params use an `Opts` struct
- [ ] Generics used only where there is immediate, concrete re-use
- [ ] `var _ Interface = (*impl)(nil)` present for every new interface implementation
- [ ] Dependencies injected as interfaces, not concrete types
- [ ] stdlib interfaces used rather than custom equivalents
- [ ] All error returns use `fmt.Errorf("...: %w", err)`
- [ ] Public constructor is `New(*Opts) Interface`; struct fields are unexported
- [ ] No naked returns anywhere in the change
- [ ] `context.Context` is the first parameter of all I/O functions; not stored in structs
