# Kerberos — Agent Instructions

## Design Skill

Before preparing **any** implementation in this repository, read the design skill:

```bash
.github/skills/design/SKILL.md
```

It defines the mandatory design rules for all Kerberos Go code (DRY, KISS, Opts
structs, generics, interface patterns, error handling, constructors, and context
propagation) and points you to the `docs/` folder for system-level understanding.

## Documentation

Keep `docs/` up to date in the same change as the code. Whenever a change
alters something a user can see (a config key, default, environment variable,
endpoint, status code, header or CLI flag), update the page that documents it.
[`docs/dev/README.md`](docs/dev/README.md#keeping-the-docs-current) lists which
page owns what. Add new pages to the index in `docs/README.md`.

When writing docs:

- Use British spelling and simple, plain language. Avoid jargon.
- Describe each setting on one page only. Elsewhere, link to it instead of
  repeating it, unless the instruction is trivial.
- Describe how Kerberos works and how to use it. Do not write about
  limitations, bugs or weaknesses.
- Do not add tests that check the docs against the code.

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

## GitHub Pages site

`pages/` holds the Angular app published to GitHub Pages (mainly the rendered API reference).
It has its own rules in [`pages/AGENTS.md`](pages/AGENTS.md) and its own `make` targets; read
those before changing anything under `pages/`. Changes to `openapi/` reach the site automatically.

## Agents

Custom multi-step agents are defined in `.github/agents/`:

| Agent | Purpose |
| --- | --- |
| `oas-updater` | Full lifecycle of adding/modifying an API endpoint |
| `testing-and-validation` | Runs all CI checks locally and reports results |
