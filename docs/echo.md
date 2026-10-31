# Echo test backend

Echo is a small test service that answers every request with a JSON description of what it received. Put it behind Kerberos to see exactly what Kerberos forwards to a backend: the path after the gateway prefix has been removed, the headers Kerberos added (such as identity headers and trace context) and the request body.

Use echo for trying Kerberos out, for testing your configuration and for troubleshooting. For real traffic, put your own services behind Kerberos.

## Running echo

### With krbctl

`krbctl compose --echo` adds an `echo` service to the generated `compose.yaml`, and `krbctl config --echo-backend` registers it as the backend `echo`. See [krbctl](./krbctl.md). The [Getting started](./getting-started.md) tutorial uses this setup.

### With Docker

The image is `ghcr.io/trebent/kerberos/echo`, with the same tags as the Kerberos image (see [Deployment: container images](./deployment.md#container-images)):

```bash
docker run --rm -p 15000:15000 ghcr.io/trebent/kerberos/echo:latest
```

### From source

From a checkout of the repository:

```bash
make echo/run
```

This runs echo on port 15000. If you have no OpenTelemetry collector running, start it with `OBSERVABILITY_ENABLED=false make echo/run`.

## Registering echo as a backend

Add echo to `gateway.router.backends` like any other backend (see [Routing](./routing.md)):

```json
{
  "gateway": {
    "router": {
      "backends": [
        { "name": "echo", "host": "echo", "port": 15000 }
      ]
    }
  }
}
```

Use `"host": "localhost"` when echo and Kerberos run directly on your machine, and the Compose service name (here `echo`) when they run in Docker Compose.

## What echo returns

Echo answers every method and path with `200 OK`, `Content-Type: application/json` and the header `X-Echo-Server: true`. The body describes the request it received:

```bash
curl -i -X POST http://localhost:30000/gw/backend/echo/hi \
  -H 'Content-Type: application/json' \
  -d '{"name":"kerberos"}'
```

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

| Field | Contents |
| --- | --- |
| `method` | The HTTP method. |
| `url` | The path echo received. Through Kerberos, this is the path after `/gw/backend/<name>` has been removed. |
| `headers` | Every request header, each with a list of values. |
| `body` | The request body, when there is one. Send JSON bodies. |

## Configuration

Echo is configured with environment variables:

| Variable | Default | Description |
| --- | --- | --- |
| `PORT` | `15000` | Port echo listens on. |
| `LOG_TO_CONSOLE` | `true` | `true` for human-readable logs, `false` for JSON logs. |
| `LOG_VERBOSITY` | `0` | Log detail. `20` also logs request and response bodies. |
| `OBSERVABILITY_ENABLED` | `true` | Turns OpenTelemetry tracing and metrics on or off. Where the data is sent is set with the standard `OTEL_*` variables, the same as for Kerberos (see [Observability](./observability.md#exporters-and-environment-variables)). |
| `TLS_CERT_FILE` | not set | Path to a PEM server certificate. Echo serves HTTPS (TLS 1.2 or newer) when this and `TLS_KEY_FILE` are both set. |
| `TLS_KEY_FILE` | not set | Path to the PEM private key for `TLS_CERT_FILE`. |
| `TLS_CLIENT_CA_FILE` | not set | Path to a PEM CA bundle. When set, together with the two TLS variables, echo asks for a client certificate and checks it against this CA (mutual TLS). |
| `VERSION` | `unset` | Version shown in logs and telemetry. The image sets it to the image tag. |

### Example: echo over HTTPS and mutual TLS

These two services test [backend TLS and mutual TLS](./tls.md#backend-tls-and-mutual-tls) from Kerberos:

```yaml
services:
  tls-echo:
    image: ghcr.io/trebent/kerberos/echo:latest
    environment:
      - TLS_CERT_FILE=/certs/tls_echo_server.pem
      - TLS_KEY_FILE=/certs/tls_echo_server.key
    volumes:
      - ./certs:/certs:ro

  mtls-echo:
    image: ghcr.io/trebent/kerberos/echo:latest
    environment:
      - TLS_CERT_FILE=/certs/mtls_echo_server.pem
      - TLS_KEY_FILE=/certs/mtls_echo_server.key
      - TLS_CLIENT_CA_FILE=/certs/ca.crt
    volumes:
      - ./certs:/certs:ro
```

See [TLS: example](./tls.md#example-certificates-mounted-as-files) for the matching Kerberos configuration.

## Related docs

- [Getting started](./getting-started.md): a first run with echo behind Kerberos
- [Routing](./routing.md): what Kerberos forwards to a backend
- [Authentication](./authentication.md): the `X-Krb-*` identity headers echo shows
- [TLS](./tls.md): HTTPS and mutual TLS to backends
- [krbctl](./krbctl.md): generating a Compose setup that includes echo
