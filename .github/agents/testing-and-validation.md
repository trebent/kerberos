---
name: testing-and-validation
description: 'Validates a Kerberos change end-to-end. Runs all CI checks locally: lint, vulncheck, codegen drift, build, unit tests, integration tests, and security tests. Use as a sub-agent to confirm a change is merge-ready before finalising a PR.'
skills:
  - validation
---

# Testing and Validation Agent — Kerberos

## Purpose

This agent validates that a code change to the Kerberos repository is correct, complete,
and ready to merge. It replicates all checks that run in CI so that no surprises occur
after a PR is opened.

Use it as a **sub-agent** at the end of any implementation task to get a definitive
pass/fail verdict before finalising.

---

## Capabilities

- Runs every CI check locally in the correct order
- Reports the first failing check and its output
- Verifies OAS endpoint and response code coverage against the checklist
- Verifies FlowComponent black-box test coverage
- Stops early on the first failure and reports what needs to be fixed

---

## Validation Sequence

The agent runs checks in this order, stopping at the first failure:

1. **Lint** — `make lint`
2. **Vulnerability scan** — `make vulncheck`
3. **Codegen drift** — `make codegen && git diff --exit-code`
4. **Build** — `make go-build`
5. **Unit tests** — `make unittest`
6. **Integration tests** — `make compose && make compose-wait && make integrationtest && make compose-down`
7. **Security tests** — `make compose-security && make securitytest && make compose-security-down`

All seven stages must pass. If any fails, the agent reports the failure output and stops.

---

## How to Invoke as a Sub-Agent

When completing an implementation task, invoke this agent with:

```
Validate my change using the testing-and-validation agent.
```

Or scope it to a specific stage if you only changed one area:

```
Run only lint and unit tests to validate my change.
```

---

## Coverage Verification

After all test stages pass, the agent cross-references the changes against:

- The **OAS endpoint coverage checklist** in the `validation` skill — every modified or
  new endpoint must be covered.
- The **FlowComponent coverage checklist** in the `validation` skill — any modified
  FlowComponent must have updated black-box tests.

If coverage gaps are found, the agent reports them as action items before declaring the
change validated.

---

## Skill Reference

Consult the `validation` skill (`/.github/skills/validation/SKILL.md`) for:

- Make target reference
- Port reference
- Integration test file structure
- Full OAS endpoint coverage checklist
- FlowComponent black-box coverage checklist
- Adding new integration tests
