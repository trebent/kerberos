# krbctl

`krbctl` is a command-line tool that scaffolds a Kerberos deployment. It asks a few questions, or reads flags, and writes a Docker Compose file (`compose.yaml`), a Kerberos configuration file (`krb.json`) and, if you choose them, the configuration for a bundled observability stack and the [admin-connector](./admin-connector.md). Use it to get a working local or test deployment in minutes, then edit the generated files for your environment.

## Installation

### From a GitHub release (recommended)

Each Kerberos GitHub release has prebuilt `krbctl` archives attached:

| OS | Architectures | Archive name |
| --- | --- | --- |
| Linux | `amd64`, `arm64` | `krbctl_<tag>_linux_<arch>.tar.gz` |
| macOS | `amd64`, `arm64` | `krbctl_<tag>_darwin_<arch>.tar.gz` |
| Windows | `amd64`, `arm64` | `krbctl_<tag>_windows_<arch>.zip` |

`<tag>` is the release tag. Each archive contains one directory with the same name as the archive, holding the `krbctl` binary (`krbctl.exe` on Windows). The release also has a `checksums.txt` file with the SHA-256 checksum of every archive.

Example for Linux on amd64:

```bash
TAG=<release tag>
ARCHIVE=krbctl_${TAG}_linux_amd64.tar.gz

curl -fsSLO https://github.com/trebent/kerberos/releases/download/${TAG}/${ARCHIVE}
curl -fsSLO https://github.com/trebent/kerberos/releases/download/${TAG}/checksums.txt

# Check the archive against checksums.txt
grep " ${ARCHIVE}\$" checksums.txt | sha256sum -c -
# On macOS: grep " ${ARCHIVE}\$" checksums.txt | shasum -a 256 -c -

tar -xzf ${ARCHIVE}
sudo install krbctl_${TAG}_linux_amd64/krbctl /usr/local/bin/krbctl
krbctl --version
```

On Windows, check the `.zip` file with `Get-FileHash <archive> -Algorithm SHA256`, compare the hash with the matching line in `checksums.txt`, then extract the archive and put `krbctl.exe` on your `PATH`.

### With `go install`

If you have Go installed:

```bash
go install github.com/trebent/kerberos/cmd/krbctl@latest
```

A binary built this way reports its version as `unset`.

### From source with `make` (Linux only)

From a checkout of the repository:

| Target | Result |
| --- | --- |
| `make krbctl/build` | Builds `build/krbctl`. |
| `make krbctl/install` | Installs `krbctl` into your Go bin directory. |

Both targets build a Linux binary (`GOOS=linux`). On macOS or Windows, use a release archive or `go install`. The version is taken from the `VERSION` make variable, for example `make krbctl/build VERSION=v1.2.3`.

## Commands

| Command | What it does |
| --- | --- |
| `krbctl compose` | Writes `compose.yaml`. |
| `krbctl config` | Writes `krb.json`, plus observability stack and admin-connector files if you choose them. |
| `krbctl completion <bash\|zsh\|fish\|powershell>` | Prints a shell completion script. See `krbctl completion <shell> --help` for how to load it. |
| `krbctl --version` (`-v`) | Prints the version. |
| `krbctl help [command]`, `-h` | Prints help. |

Both `compose` and `config` are interactive by default. Pass `-y` (`--non-interactive`) to skip the questions and use flags instead. You usually run both commands into the same directory, because `compose.yaml` mounts the files that `config` writes using paths relative to itself (`./krb.json`, `./connector.json`, `./prometheus.yml`, `./grafana/...`, `./jaeger.yml`, `./jaeger-config-ui.json`, `./loki.yml`).

## `krbctl compose`

Writes `<output>/compose.yaml`.

### Interactive mode

`krbctl compose` asks four yes/no questions, all on one screen:

1. Include the echo service? (a [test backend](./echo.md) that replies with the request it received)
2. Include the observability stack? (Prometheus, Grafana, Jaeger and Loki)
3. Include PostgreSQL as the persistence backend? (SQLite is used if you answer no)
4. Include the admin-connector service?

If you include the admin-connector without the observability stack, a second screen asks which tool it should protect, as `host:port` inside the Compose network.

### Flags

| Flag | Default | Description |
| --- | --- | --- |
| `-o`, `--output` | `.` | Directory to write `compose.yaml` to. It must already exist. |
| `-y`, `--non-interactive` | `false` | Skip the questions and use the flags below. |
| `--echo` | `false` | Include the echo service. Only used with `-y`. |
| `--obs-stack` | `false` | Include Prometheus, Grafana, Jaeger and Loki. Only used with `-y`. |
| `--postgres` | `false` | Use PostgreSQL instead of SQLite. Only used with `-y`. |
| `--connector` | `false` | Include the admin-connector. Only used with `-y`. |
| `--connector-target` | `jaeger:16686` with `--obs-stack`, otherwise none | `host:port` (or `host`) the admin-connector forwards to, without a scheme or path. Required with `--connector` when `--obs-stack` is not set. Only used with `-y`. |

Without `-y`, the `--echo`, `--obs-stack`, `--postgres`, `--connector` and `--connector-target` flags are ignored and you are asked the questions.

### Generated services

| Service | Image | Included when | Published ports | Volumes and mounts |
| --- | --- | --- | --- | --- |
| `kerberos` | `ghcr.io/trebent/kerberos:latest` | Always | `30000` (gateway), `30001` (admin API) | `./krb.json` (read-only); `krbdata` volume at `/krbdata` with SQLite |
| `sqlite-init` | `busybox:1.38` | PostgreSQL not chosen | None | `krbdata` volume. Creates `/krbdata/krb.db` with permissions every service can use, then exits. |
| `postgres` | `postgres:18.4-alpine3.23` | PostgreSQL chosen | None | `postgres` volume. Database, user and password are all `kerberos`. Has a health check that `kerberos` waits for. |
| `echo` | `ghcr.io/trebent/kerberos/echo:latest` | Echo chosen | None (reachable inside the Compose network as `echo:15000`) | None |
| `connector` | `ghcr.io/trebent/kerberos/admin-connector:latest` | Connector chosen | `30100` | `./connector.json` (read-only); `krbdata` volume with SQLite |
| `prometheus` | `prom/prometheus:v3` | Observability stack chosen | None | `./prometheus.yml`; `prometheus` volume. Data is limited to 1 GB. |
| `grafana` | `grafana/grafana:13.1` | Observability stack chosen | `3000` | `./grafana/*` files; `grafana` volume; `krbdata` volume with SQLite |
| `jaeger-init` | `busybox:1.38` | Observability stack chosen | None | `jaeger` volume. Sets ownership for Jaeger, then exits. |
| `jaeger` | `jaegertracing/jaeger:2.20.0` | Observability stack chosen | `16686` (UI), only if the connector is not chosen | `./jaeger.yml`, `./jaeger-config-ui.json`; `jaeger` volume |
| `loki` | `grafana/loki:3.7` | Observability stack chosen | `3100`, on `127.0.0.1` only, so only your own machine can reach it | `./loki.yml`; `loki` volume |

All services use `restart: on-failure`. `kerberos` runs with `LOG_TO_CONSOLE=1`, `PORT=30000` and `ADMIN_PORT=30001`. See [Deployment](./deployment.md) for what these variables do.

### How the options interact

- **Observability stack.** If you choose it, `kerberos`, `echo` and `connector` send traces with OTLP over gRPC to `http://jaeger:4317` and expose Prometheus metrics on port `9464` in the Compose network. If you do not choose it, those services get `OTEL_METRICS_EXPORTER=none` and `OTEL_TRACES_EXPORTER=none`. See [Observability](./observability.md).
- **Jaeger and the connector.** If you choose both, Jaeger's UI is not published on the host. You reach it only through the admin-connector at `http://localhost:30100`, which needs a Kerberos admin login (see [Admin connector](./admin-connector.md)). If you choose the observability stack without the connector, the Jaeger UI is published directly on `http://localhost:16686` with no login.
- **Connector target.** The connector's `TARGET` comes from `--connector-target` or the interactive question. With the observability stack it defaults to the Jaeger UI at `jaeger:16686`; without it, you must give one. See [Admin connector](./admin-connector.md).
- **Shared database.** `kerberos`, `connector` and `grafana` use the same database: the `/krbdata/krb.db` SQLite file on the `krbdata` volume, or the `kerberos` PostgreSQL database. Grafana keeps its tables next to the Kerberos tables.

### Container logs

With the observability stack, every service except `loki` sends its output to Loki, and you can search it in Grafana with the Loki data source. Each log line is labelled with the service it came from, for example `{compose_service="kerberos"}`. `docker compose logs` keeps working as usual.

Logs are sent by the Loki Docker logging plugin. Install it once on the machine that runs Docker, before you run `docker compose up`:

```bash
docker plugin install grafana/loki-docker-driver:3.7.9-amd64 --alias loki --grant-all-permissions
```

On an ARM machine, use the `3.7.9-arm64` tag. The plugin runs as part of Docker, outside the Compose network, so it sends logs to Loki on `127.0.0.1:3100`. Every service waits for `loki` to start, and Compose stops `loki` last.

## `krbctl config`

Writes `<output>/krb.json`, and optionally the observability stack and admin-connector files.

### Interactive mode

The questions come in this order:

1. **Use the echo service as a backend?** If yes, a backend named `echo` at `echo:15000` is added.
2. **Backends.** For each backend you enter a name, a host (default `localhost`) and a port (default `8080`), then answer "Add another backend?". Leave the name empty and press Enter to finish. At least one backend is required, counting `echo`. A name that is already used, or a port that is not a number between 1 and 65535, is rejected and asked again.
3. **A final form** with these sections. Press Shift+Tab to go back to an earlier section.
   - **Persistence:** SQLite (default) or PostgreSQL.
   - **Authentication:** for each backend, whether to enable basic auth.
   - **Observability stack:** whether to generate the Prometheus, Grafana, Jaeger and Loki files. If yes, you also choose the Prometheus scrape targets (Kerberos, Jaeger, Loki and all backends are selected by default) and whether Grafana allows anonymous access (default yes).
   - **Admin-connector:** whether to generate `connector.json`. If yes, you also choose "Allow all CORS origins?" (yes = allow all, no = deny all cross-origin requests).

Interactive mode runs in a terminal. If you cancel a prompt (for example with Ctrl+C), the command stops and writes nothing. In scripts and CI, use `-y`.

### Flags

| Flag | Default | Description |
| --- | --- | --- |
| `-o`, `--output` | `.` | Directory to write files to. It must already exist. |
| `-y`, `--non-interactive` | `false` | Skip the questions and use the flags below. With `-y`, at least one of `--echo-backend` or `--backend` is required. |
| `--echo-backend` | `false` | Add the echo service as a backend named `echo` at `echo:15000` (without auth). |
| `--backend` | none | Add a backend. Repeatable. See the syntax below. |
| `--driver` | `sqlite` | Persistence driver: `sqlite` or `postgres`. |
| `--obs-stack` | `false` | Generate the observability stack files. |
| `--grafana-anonymous` | `true` | Let anyone view Grafana dashboards without logging in. Use `--grafana-anonymous=false` to turn it off. |
| `--scrape-target` | Kerberos, Jaeger, Loki and all backends | Prometheus scrape target. Repeatable. If you pass any, only those are used. |
| `--connector` | `false` | Generate `connector.json` for the admin-connector. |
| `--connector-allow-all-origins` | `true` | `true` writes `"allowAll": true`. `false` writes `"denyAll": true`. |

All flags except `-o` and `-y` are only used with `-y`. Without `-y` you are asked the questions instead.

#### `--backend` syntax

```text
--backend 'name=<name>,host=<host>,port=<port>,auth=<true|false>'
```

| Key | Required | Default | Notes |
| --- | --- | --- | --- |
| `name` | Yes | | Backend name. Requests reach it at `/gw/backend/<name>/...` on the gateway. See [Routing](./routing.md). |
| `host` | No | `localhost` | Host name as seen from the Kerberos container. |
| `port` | No | `8080` | Port the backend listens on, 1–65535. Any other value stops `krbctl` with an error. |
| `auth` | No | `false` | `true` puts the backend behind basic authentication. |

Backend names must be unique, including against `echo` when you use `--echo-backend`. A duplicate stops `krbctl` with an error.

#### `--scrape-target` values

| Value | Scraped address |
| --- | --- |
| `kerberos` | `kerberos:9464` |
| `jaeger` | `jaeger:8888` |
| `loki` | `loki:3100` |
| A backend name | `<backend host>:9464` |

When the connector is enabled, `connector:9464` is always added and does not need to be listed.

### What `krb.json` contains

- `gateway.router.backends`: one entry per backend with `name`, `host` and `port`.
- `observability`: always `{"enabled": true, "runtimeMetrics": true}`. Whether anything is exported depends on the `OTEL_*` variables in `compose.yaml`.
- `auth`: only if at least one backend has auth enabled. It enables the `basic` method, adds one mapping per protected backend with `method: "basic"` and an empty `exempt` list, and sets `order: 1`. See [Authentication](./authentication.md).
- `persistence`:
  - SQLite: `{"driver": "sqlite", "address": "/krbdata/krb.db"}`
  - PostgreSQL: `{"driver": "postgres", "address": "postgres:5432", "postgres": {"database": "kerberos", "username": "kerberos", "password": "kerberos", "sslMode": "disable"}}`

`krb.json` has no `admin` section, so the superuser is created with the [default credentials](./admin-api.md#the-superuser). See [What to change afterwards](#what-to-change-afterwards).

Example from `krbctl config -y --echo-backend`:

```json
{
  "gateway": {
    "router": {
      "backends": [
        {
          "host": "echo",
          "name": "echo",
          "port": 15000
        }
      ]
    }
  },
  "observability": {
    "enabled": true,
    "runtimeMetrics": true
  },
  "persistence": {
    "address": "/krbdata/krb.db",
    "driver": "sqlite"
  }
}
```

### Other generated files

| Path (relative to `-o`) | Generated when | Contents |
| --- | --- | --- |
| `prometheus.yml` | Observability stack | Scrapes every 15 seconds, one job per scrape target. |
| `grafana/grafana.ini` | Observability stack | Grafana database settings (the same database as Kerberos) and anonymous access (`Viewer` role in `Main Org.` when enabled). |
| `grafana/grafana-datasources.yml` | Observability stack | A Prometheus data source at `http://prometheus:9090` and a Loki data source at `http://loki:3100`. |
| `grafana/grafana-dashboards.yml` | Observability stack | Loads the five dashboards below. |
| `grafana/prometheus.json` | Observability stack | Dashboard "Prometheus status". |
| `grafana/kerberos_runtime.json` | Observability stack | Dashboard "Kerberos runtime". |
| `grafana/kerberos_http.json` | Observability stack | Dashboard "Kerberos HTTP". |
| `grafana/jaeger.json` | Observability stack | Dashboard "Jaeger". |
| `grafana/loki.json` | Observability stack | Dashboard "Loki". |
| `jaeger.yml` | Observability stack | Jaeger with an OTLP gRPC receiver on `4317`, UI on `16686`, its own metrics on `8888`, and Badger storage on the `jaeger` volume (traces kept for 48 hours, archived traces for 72 hours). |
| `jaeger-config-ui.json` | Observability stack | Jaeger UI settings (archiving enabled). |
| `loki.yml` | Observability stack | Loki as a single process with its data on the `loki` volume. It accepts logs without a login. |
| `connector.json` | Connector | `origins` (`allowAll` or `denyAll`) and the same `persistence` block as `krb.json`. See [Admin connector](./admin-connector.md). |

`krbctl config` creates the `grafana` directory if needed and replaces existing files.

## Give the same answers to both commands

`compose` and `config` do not share state. The generated files only work together if you give the same answers to both:

| `krbctl compose` | `krbctl config` |
| --- | --- |
| `--echo` / "Include the echo service?" | `--echo-backend` / "Use the echo service as a backend?" |
| `--postgres` / "Include PostgreSQL...?" | `--driver postgres` / Persistence: PostgreSQL |
| `--obs-stack` / "Include the observability stack?" | `--obs-stack` / "Generate observability stack config?" |
| `--connector` / "Include the admin-connector service?" | `--connector` / "Include the admin-connector?" |

For example, with `--postgres` on both commands, `krb.json` points to `postgres:5432` and `compose.yaml` runs the `postgres` service.

Backends you add with `--backend` (or in the interactive backend questions) are only written to `krb.json`. `krbctl compose` does not add services for them. Add them to `compose.yaml` yourself, or set `host` to an address the Kerberos container can reach. Inside the container, `localhost` is the container itself, not your machine.

## Examples

### Minimal: gateway and echo backend with SQLite

```bash
mkdir kerberos-demo && cd kerberos-demo
krbctl compose -y --echo
krbctl config -y --echo-backend
docker compose up -d

curl -i http://localhost:30000/gw/backend/echo/hello
```

This starts `sqlite-init`, `kerberos` and [`echo`](./echo.md). Metrics and traces are not exported. See [Getting started](./getting-started.md) for a full first-run walkthrough.

### Full stack: PostgreSQL, observability and admin-connector

```bash
mkdir kerberos-full
krbctl compose -y -o kerberos-full --echo --postgres --obs-stack --connector
krbctl config  -y -o kerberos-full --echo-backend \
  --backend 'name=orders,host=orders,port=8000,auth=true' \
  --driver postgres --obs-stack --connector
cd kerberos-full && docker compose up -d
```

Install the Loki logging plugin first, as described in [Container logs](#container-logs).

`krbctl config` writes:

```text
kerberos-full/compose.yaml          (from krbctl compose)
kerberos-full/krb.json
kerberos-full/prometheus.yml
kerberos-full/grafana/grafana.ini
kerberos-full/grafana/grafana-datasources.yml
kerberos-full/grafana/grafana-dashboards.yml
kerberos-full/grafana/prometheus.json
kerberos-full/grafana/kerberos_runtime.json
kerberos-full/grafana/kerberos_http.json
kerberos-full/grafana/jaeger.json
kerberos-full/grafana/loki.json
kerberos-full/jaeger.yml
kerberos-full/jaeger-config-ui.json
kerberos-full/loki.yml
kerberos-full/connector.json
```

After starting:

| What | Where |
| --- | --- |
| Gateway | `http://localhost:30000/gw/backend/<backend>/...` |
| Admin API | `http://localhost:30001/api/admin/...` |
| Grafana | `http://localhost:3000` |
| Jaeger UI (through the admin-connector) | `http://localhost:30100`, after an admin login on `:30001` |

Log in as the superuser so the connector lets you through. The session cookie is shared with the connector because both run on `localhost`:

```bash
curl -i -c cookies.txt -X POST http://localhost:30001/api/admin/superuser/login \
  -H 'Content-Type: application/json' \
  -d '{"clientId":"admin","clientSecret":"secret"}'
```

In this example, the `orders` backend is protected with basic auth but has no service in `compose.yaml`. Add an `orders` service listening on port `8000`, or Kerberos cannot reach it. Its Prometheus job (`orders:9464`) only gets data if the service exposes metrics on that port.

## What to change afterwards

The generated files are a starting point for local and test setups. Before using them for anything else:

- **Change the superuser secret.** See [Admin API: the superuser](./admin-api.md#the-superuser).
- **Change the database passwords.** PostgreSQL uses `kerberos` / `kerberos` in `compose.yaml`, `krb.json`, `connector.json` and `grafana/grafana.ini`, with `sslMode: disable`. See [Persistence](./persistence.md).
- **Point to real backends.** Edit `gateway.router.backends` in `krb.json` and add auth mappings as needed. See [Routing](./routing.md) and [Authentication](./authentication.md).
- **Enable TLS.** Login cookies need HTTPS on any host other than `localhost`. See [TLS](./tls.md) and [Sessions](./sessions.md).
- **Pin image versions.** All Kerberos images use the `latest` tag. See [Deployment](./deployment.md) for available tags.
- **Restrict the connector's origins.** Replace `allowAll` with the origins you use. See [Admin connector](./admin-connector.md).
- **Review Grafana access.** Turn off anonymous viewing if you do not want it, and change the password of Grafana's own admin account. See [Observability](./observability.md).

For all configuration keys, see [Configuration](./configuration.md).

## Related docs

- [Getting started](./getting-started.md)
- [Echo test backend](./echo.md)
- [Configuration](./configuration.md)
- [Deployment](./deployment.md)
- [Persistence](./persistence.md)
- [Observability](./observability.md)
- [Admin connector](./admin-connector.md)
- [Admin API](./admin-api.md)
