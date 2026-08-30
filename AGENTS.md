# Kerberos — Agent Instructions

## Design Skill

Before preparing **any** implementation in this repository, read the design skill:

```bash
.github/skills/design/SKILL.md
```

It defines the mandatory design rules for all Kerberos Go code (DRY, KISS, Opts
structs, generics, interface patterns, error handling, constructors, and context
propagation) and points you to the `docs/` folder for system-level understanding.

## Available Skills

The following skills live in `.github/skills/` and cover specific areas of the
codebase. Use them when working in the relevant area:

| Skill | When to use |
| --- | --- |
| `design` | Before any implementation — read first |
| `configuration` | Adding or modifying config sections or schemas |
| `flow-components` | Implementing or debugging FlowComponents |
| `oas` | Adding or modifying OpenAPI specifications |
| `validation` | Validating a change end-to-end before merging |

## Agents

Custom multi-step agents are defined in `.github/agents/`:

| Agent | Purpose |
| --- | --- |
| `oas-updater` | Full lifecycle of adding/modifying an API endpoint |
| `testing-and-validation` | Runs all CI checks locally and reports results |
