# krbctl

`krbctl` is the Kerberos setup CLI. It helps you generate a baseline deployment so you can start running Kerberos quickly and then refine it.

## What it helps with

- Getting started with a local Kerberos deployment layout.
- Generating a base `compose.yaml`.
- Generating a base Kerberos config (`krb.json`).
- Optionally generating related files for bundled services (observability stack and admin-connector config).

## Commands

### `krbctl compose`

Interactive command that writes a baseline `compose.yaml`.

It can include or skip optional services such as:

- `echo`
- observability stack (Prometheus, Grafana, Jaeger)
- PostgreSQL
- admin-connector

### `krbctl config`

Interactive command that writes a baseline `krb.json` and, when selected, related files:

- observability stack files (`prometheus.yml`, Grafana provisioning/dashboard files, Jaeger config)
- `connector.json` for the admin-connector

The prompt flow lets you define backend targets, choose persistence mode, and toggle optional auth and observability stack generation.

## Typical workflow

1. Run `krbctl compose` to scaffold deployment services.
2. Run `krbctl config` to scaffold Kerberos configuration files.
3. Adjust generated files for your environment.
4. Start services with your compose workflow.

For implementation details and source, see `cmd/krbctl/`.
