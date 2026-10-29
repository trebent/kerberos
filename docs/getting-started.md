# Getting started

This tutorial gets a Kerberos gateway running on your machine, sends a request through it to a test backend, and walks through the first logins: the superuser on the Admin API, and a user of an organisation calling a protected backend. Choose one of two paths:

- **Path A (recommended):** generate a Docker Compose setup with `krbctl` and run the published images.
- **Path B:** run Kerberos and the test backend from source with Go and `make`.

Both paths use the default ports: **30000** for the gateway (backend traffic) and **30001** for the admin server (Admin API and organisation API).

## Prerequisites

| Path | You need |
| --- | --- |
| A | Docker with the Compose plugin (`docker compose`), `curl` |
| B | Go (the version in `go.mod`), `make`, `git`, `curl` |

## The echo test backend

Both paths put a small test service called **echo** behind the gateway. Echo answers every request with a JSON description of what it received, so you can see exactly what Kerberos forwarded: the path after the gateway prefix was stripped, the headers Kerberos added (identity headers, trace context), and the request body.

```json
{
  "method": "POST",
  "url": "/hi",
  "headers": {
    "Content-Type": ["application/json"],
    "Traceparent": ["00-36efcf49b4f889c5fd1bedf7077584bf-8321f1aa71e3cbb9-01"],
    "X-Krb-Org": ["1"],
    "X-Krb-User": ["2"]
  },
  "body": {"name": "kerberos"}
}
```

Every response also carries the header `X-Echo-Server: true`.

| Item | Value |
| --- | --- |
| Image | `ghcr.io/trebent/kerberos/echo` (tags `latest` and release tags) |
| Listen port | `PORT`, default `15000` |
| `LOG_TO_CONSOLE` | Human-readable logs (default `true`) |
| `LOG_VERBOSITY` | Log verbosity, default `0`; `20` also logs request and response bodies |
| `OBSERVABILITY_ENABLED` | OpenTelemetry tracing and metrics, default `true`; exporters are configured with the standard `OTEL_*` variables |
| `TLS_CERT_FILE`, `TLS_KEY_FILE` | Serve HTTPS (TLS 1.2 or newer) when both are set |
| `TLS_CLIENT_CA_FILE` | Require and verify client certificates (mutual TLS); needs the two TLS variables above |
| `VERSION` | Version string reported in logs and telemetry |

Things to know about echo:

- Request bodies must be JSON. A non-JSON body (for example `curl -d 'plain text'`) makes echo answer `500`.
- Echo is a testing and debugging tool. Do not use it as a real backend.

## Path A: krbctl and Docker Compose

### 1. Install krbctl

`krbctl` is a small CLI that generates a `compose.yaml` and a Kerberos configuration file. Each [GitHub release](https://github.com/trebent/kerberos/releases) has these assets:

- `krbctl_<tag>_linux_amd64.tar.gz`, `krbctl_<tag>_linux_arm64.tar.gz`
- `krbctl_<tag>_darwin_amd64.tar.gz`, `krbctl_<tag>_darwin_arm64.tar.gz`
- `krbctl_<tag>_windows_amd64.zip`, `krbctl_<tag>_windows_arm64.zip`
- `checksums.txt` (SHA-256 checksums of all archives)

Each archive holds a directory named after the archive, with the `krbctl` binary (`krbctl.exe` on Windows) inside. For example, on Linux amd64:

```bash
TAG=<release tag>          # for example the latest tag on the releases page
ASSET=krbctl_${TAG}_linux_amd64

curl -fsSLO "https://github.com/trebent/kerberos/releases/download/${TAG}/${ASSET}.tar.gz"
curl -fsSLO "https://github.com/trebent/kerberos/releases/download/${TAG}/checksums.txt"
sha256sum --check --ignore-missing checksums.txt

tar -xzf "${ASSET}.tar.gz"
sudo install "${ASSET}/krbctl" /usr/local/bin/krbctl
krbctl --version
```

If you have Go installed you can also run `go install github.com/trebent/kerberos/cmd/krbctl@latest`. That build works the same, but `krbctl --version` reports `unset`.

### 2. Generate the files

Create an empty directory and run both generators in it. The output directory (`-o`, default `.`) must already exist.

```bash
mkdir kerberos-quickstart && cd kerberos-quickstart

# compose.yaml with Kerberos, the echo service and SQLite storage
krbctl compose -y --echo

# krb.json with echo registered as backend "echo", protected by basic auth
krbctl config -y --backend name=echo,host=echo,port=15000,auth=true
```

`-y` (`--non-interactive`) builds the files from flags. Without `-y`, both commands ask their questions in an interactive terminal form instead. In `krbctl config` you can press Shift+Tab to go back to an earlier section.

About the flags used here:

- `krbctl config --echo-backend` is a shorter way to register echo, but it always registers it **without** authentication. This tutorial uses `--backend ...,auth=true` instead, so that it can show the login flow. If you only want to see traffic flow, use `--echo-backend` and skip [step 5](#5-log-in-as-the-superuser) onwards.
- The two commands do not talk to each other. If you add `--postgres`, `--obs-stack` or `--connector` to `krbctl compose`, give the matching `--driver postgres`, `--obs-stack` or `--connector` to `krbctl config` too. See [krbctl](./krbctl.md) for every flag and the files they generate.

The generated `krb.json` looks like this:

```json
{
  "auth": {
    "methods": {
      "basic": {}
    },
    "order": 1,
    "scheme": {
      "mappings": [
        {
          "backend": "echo",
          "exempt": [],
          "method": "basic"
        }
      ]
    }
  },
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

The generated `compose.yaml` runs:

| Service | What it does |
| --- | --- |
| `sqlite-init` | One-off container that prepares the SQLite database file on the `krbdata` volume |
| `kerberos` | `ghcr.io/trebent/kerberos:latest` with `./krb.json` mounted, ports 30000 and 30001 published |
| `echo` | `ghcr.io/trebent/kerberos/echo:latest` on port 15000, **not** published: it is only reachable through Kerberos |

Without the observability stack, telemetry export is switched off with `OTEL_METRICS_EXPORTER=none` and `OTEL_TRACES_EXPORTER=none`.

### 3. Start the stack

```bash
docker compose up -d
docker compose ps
```

Check that the gateway is up. The health endpoint answers `204 No Content`:

```bash
curl -i http://localhost:30000/gw/health
```

### 4. Send a request through the gateway

Backends are addressed as `/gw/backend/<backend name>/<path>`. Kerberos removes the `/gw/backend/echo` prefix and forwards the rest of the path to the backend.

```bash
curl -i http://localhost:30000/gw/backend/echo/hello
```

Because `echo` is protected by basic auth and you have no session yet, Kerberos rejects the request before it reaches echo:

```text
HTTP/1.1 401 Unauthorized
Content-Type: application/json

{"errors":["Unauthorized"]}
```

(If you generated the config with `--echo-backend`, the request is forwarded and echo answers `200` with the JSON shown in [The echo test backend](#the-echo-test-backend).)

Note that the path needs a slash after the backend name: `/gw/backend/echo` alone returns `400`, while `/gw/backend/echo/` reaches the backend root `/`.

### 5. Log in as the superuser

Kerberos creates a **superuser** the first time it starts, with client ID `admin` and client secret `secret`. Log in on the admin port and keep the session cookies in a cookie jar file (`-c` writes the jar, `-b` sends it):

```bash
curl -i -c admin.txt -X POST http://localhost:30001/api/admin/superuser/login \
  -H 'Content-Type: application/json' \
  -d '{"clientId":"admin","clientSecret":"secret"}'
```

A successful login answers `204 No Content` and sets three cookies: `session`, `refresh` and `csrf`. Check who you are:

```bash
curl -b admin.txt http://localhost:30001/api/admin/me
# {"isSuperuser":true}
```

**Change the superuser secret now.** The `admin.superUser` credentials in the configuration (default `admin` / `secret`) are only used to create the superuser on the very first start; changing the configuration later has no effect. Change the secret through the API:

```bash
curl -i -b admin.txt -X PUT http://localhost:30001/api/admin/superuser/password \
  -H 'Content-Type: application/json' \
  -d '{"oldPassword":"secret","newPassword":"<a long random secret>"}'
```

See [Admin API](./admin-api.md) for admin users, groups and permissions.

### 6. Create an organisation and a user

Users who call protected backends belong to an **organisation**. The organisation API lives on the admin port under `/api/auth/basic`, and the superuser may use it for any organisation.

Create an organisation:

```bash
curl -s -b admin.txt -X POST http://localhost:30001/api/auth/basic/organisations \
  -H 'Content-Type: application/json' \
  -d '{"name":"acme"}'
```

```json
{"adminPassword":"7d2c3667-f782-42e9-b7a4-34037238e04f","adminUserId":1,"adminUsername":"admin-acme","id":1,"name":"acme"}
```

Kerberos also creates an administrator for the organisation, `admin-<name>`, with a random password. The password is shown **only in this response**, so store it if you want to use that account later. Note the organisation `id` (here `1`) because the next calls use it.

Create a user in the organisation. The name must be at least 5 characters and the password 10 to 40 characters:

```bash
curl -s -b admin.txt -X POST http://localhost:30001/api/auth/basic/organisations/1/users \
  -H 'Content-Type: application/json' \
  -d '{"name":"alice","password":"correct-horse-42"}'
# {"id":2,"name":"alice"}
```

See [Organisations](./organizations.md) for groups, group membership and what the organisation administrator can do.

### 7. Call the protected backend as the user

Log in as `alice` with a separate cookie jar:

```bash
curl -i -c alice.txt -X POST http://localhost:30001/api/auth/basic/organisations/1/login \
  -H 'Content-Type: application/json' \
  -d '{"username":"alice","password":"correct-horse-42"}'
```

Repeat the request from step 4, this time sending alice's session cookie:

```bash
curl -i -b alice.txt http://localhost:30000/gw/backend/echo/hello
```

This time the request reaches echo:

```text
HTTP/1.1 200 OK
Content-Type: application/json
X-Echo-Server: true

{
  "method": "GET",
  "url": "/hello",
  "headers": {
    "Accept": ["*/*"],
    "Accept-Encoding": ["gzip"],
    "Cookie": ["session=0f722185-...; csrf=49f40f41-..."],
    "Traceparent": ["00-36efcf49b4f889c5fd1bedf7077584bf-8321f1aa71e3cbb9-01"],
    "User-Agent": ["curl/8.5.0"],
    "X-Krb-Org": ["1"],
    "X-Krb-User": ["2"]
  }
}
```

(Output shortened; echo pretty-prints each header array over several lines.) Things to notice:

- `url` is `/hello`: the gateway prefix was stripped.
- Kerberos added `X-Krb-Org` and `X-Krb-User`, the numeric IDs of the caller's organisation and user. Backends use these to identify the caller. See [Authentication](./authentication.md).
- The client's headers, including `Cookie`, are forwarded unchanged.

The session lasts 15 minutes. Log out with:

```bash
curl -i -b alice.txt -X POST http://localhost:30001/api/auth/basic/organisations/1/logout
```

See [Sessions](./sessions.md) for refreshing sessions, cookie lifetimes and CSRF protection.

### 8. Stop or reset

```bash
docker compose down       # stop, keep the database volume
docker compose down -v    # stop and delete the database (next start creates admin/secret again)
```

## Path B: run from source

Use this path to try unreleased changes or to work on Kerberos itself.

```bash
git clone https://github.com/trebent/kerberos.git
cd kerberos
```

Start echo in one terminal:

```bash
make echo/run
```

This runs echo on port 15000. Echo exports telemetry by default; if you have no OpenTelemetry collector running and see export errors in its log, start it with `OBSERVABILITY_ENABLED=false make echo/run` instead.

Start Kerberos in a second terminal:

```bash
make run
```

`make run` starts the gateway on port 30000 and the admin server on port 30001, with human-readable logs at verbosity 20, the OpenAPI files from `openapi/`, and the configuration file `test/config/local.json`. That configuration:

- registers `echo` at `localhost:15000`, and a second backend `missing` at `localhost:17000` where nothing listens (requests to it return `500 {"errors":["failed to forward request"]}`);
- protects `echo` with basic auth, exactly like Path A;
- validates requests to `echo` against the OpenAPI document `test/oas/echo.yaml`, which only defines the paths `/`, `/hi` and `/long/hello`;
- stores data in SQLite at `build/krb.db`;
- disables observability.

Because of the OpenAPI validation, use `/hi` instead of `/hello` in this setup. Other paths are rejected with `404 {"errors":["no matching operation was found"]}` once you are logged in.

From here, follow [steps 5 to 7 of Path A](#5-log-in-as-the-superuser) with the path changed:

```bash
curl -i -b alice.txt http://localhost:30000/gw/backend/echo/hi

curl -i -b alice.txt -X POST http://localhost:30000/gw/backend/echo/hi \
  -H 'Content-Type: application/json' \
  -d '{"name":"kerberos"}'
```

`make krb/superuser-login` is a shortcut that performs the superuser login and prints the response headers, including the `Set-Cookie` values. To start over with an empty database, stop Kerberos and delete `build/krb.db`.

## Things to know

- **Secure cookies over plain HTTP.** All session cookies carry the `Secure` attribute. curl (7.79 and newer) and current browsers treat `http://localhost` as a secure context and still send them, which is why this tutorial works without TLS. Use the hostname `localhost` consistently: cookies stored for `localhost` are not sent to `127.0.0.1`. On any other host name you need HTTPS; see [TLS](./tls.md).
- **The admin port is a separate server.** Logins and the Admin and organisation APIs are on port 30001; backend traffic is on port 30000. Cookies are not port-specific, so a session obtained on port 30001 is sent to port 30000 too. Only organisation user sessions open protected backends; a superuser or admin-user session is rejected there with `401`.
- **One `session` cookie name for both logins.** The superuser login and the organisation login both use a cookie named `session`. In a browser on the same host, logging in to one replaces the other. With curl, keep separate cookie jars as shown above.
- **Query strings are not forwarded.** `GET /gw/backend/echo/hi?x=1` reaches echo as `/hi`. See [Routing](./routing.md).
- **Default credentials.** The superuser starts as `admin` / `secret` on every new database. Change it immediately on anything other than a throwaway local setup.

## Next steps

- [Configuration](./configuration.md): the configuration file format, environment references and all sections.
- [Routing](./routing.md): registering your own backends, timeouts and forwarding behaviour.
- [Authentication](./authentication.md): protecting backends, exempt paths and group-based authorisation.
- [Observability](./observability.md): metrics, tracing and dashboards (try `--obs-stack` with `krbctl`).
- [Deployment](./deployment.md): running Kerberos in production, environment variables and a security checklist.
- [krbctl](./krbctl.md): every `krbctl` option, including PostgreSQL, the observability stack and the admin-connector.
