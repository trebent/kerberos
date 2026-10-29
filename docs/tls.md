# TLS

Kerberos can use HTTPS on the gateway port, on the admin port, and on the connection to each backend. Backend connections can also use mutual TLS (mTLS). This page describes each setting, how certificates are loaded, what is not configurable, and how TLS affects login cookies.

- [Overview](#overview)
- [Gateway server TLS](#gateway-server-tls)
- [Admin API TLS](#admin-api-tls)
- [Backend TLS and mutual TLS](#backend-tls-and-mutual-tls)
- [Certificate loading and startup errors](#certificate-loading-and-startup-errors)
- [Secure cookies need HTTPS](#secure-cookies-need-https)
- [Running behind a TLS-terminating proxy](#running-behind-a-tls-terminating-proxy)
- [Example: certificates mounted as files](#example-certificates-mounted-as-files)
- [Admin connector](#admin-connector)
- [Limitations and things to know](#limitations-and-things-to-know)
- [Related docs](#related-docs)

## Overview

There are three separate TLS settings. Each one is optional, and each is enabled when its `tls` block is present.

| Connection | Config block | Effect when present |
| --- | --- | --- |
| Client to gateway (port `PORT`, default 30000) | `gateway.tls` | The gateway port serves HTTPS only. |
| Client to admin API (port `ADMIN_PORT`, default 30001) | `admin.api.tls` | The admin port serves HTTPS only. This includes the basic-auth API. |
| Kerberos to a backend | `gateway.router.backends[].tls` | Kerberos uses `https://` for that backend and verifies its certificate. It can also present a client certificate (mTLS). |

All certificates and keys are PEM files that Kerberos reads from disk. Relative paths are resolved against the working directory of the Kerberos process. In containers, mount the files and use absolute paths.

The authoritative definitions are the JSON schemas in `internal/config/schemas/`: `gateway_schema.json`, `admin_schema.json` and `router_schema.json`.

## Gateway server TLS

```json
{
  "gateway": {
    "tls": {
      "serverCertFile": "/certs/krb.pem",
      "serverKeyFile": "/certs/krb-key.pem"
    },
    "router": {
      "backends": [ { "name": "echo", "host": "echo", "port": 15000 } ]
    }
  }
}
```

| Key | Type | Default | Description |
| --- | --- | --- | --- |
| `gateway.tls` | object | not set (plain HTTP) | When present, the gateway port serves HTTPS only. |
| `gateway.tls.serverCertFile` | string | required | Path to the PEM server certificate. To send intermediate certificates, put the full chain in this file with the leaf certificate first. |
| `gateway.tls.serverKeyFile` | string | required | Path to the PEM private key for that certificate. |

When `gateway.tls` is set:

- Kerberos does not listen for plain HTTP or redirect it. A plain `http://` request to the gateway port gets `400 Bad Request` with the text `Client sent an HTTP request to an HTTPS server.`
- The health endpoint `/gw/health` is also HTTPS-only, so health checks and load-balancer probes must use HTTPS. See [Deployment](./deployment.md).
- Clients can use HTTP/1.1 or HTTP/2.

```bash
curl --cacert ca.pem https://localhost:30000/gw/backend/echo/hello
```

## Admin API TLS

```json
{
  "admin": {
    "api": {
      "tls": {
        "serverCertFile": "/certs/krb.pem",
        "serverKeyFile": "/certs/krb-key.pem"
      }
    }
  }
}
```

| Key | Type | Default | Description |
| --- | --- | --- | --- |
| `admin.api.tls` | object | not set (plain HTTP) | When present, the admin port serves HTTPS only. |
| `admin.api.tls.serverCertFile` | string | required | Path to the PEM server certificate (full chain, leaf first). |
| `admin.api.tls.serverKeyFile` | string | required | Path to the PEM private key. |

The admin port serves both the [admin API](./admin-api.md) and the basic-auth (organisation) API described in [Organisations](./organizations.md). One `admin.api.tls` block therefore covers both. `auth.methods.basic.api` has no separate TLS setting.

Gateway TLS and admin TLS are independent. You can use the same certificate for both if it is valid for every host name clients use. You can also enable TLS on only one of the two ports.

```bash
curl --cacert ca.pem -X POST https://localhost:30001/api/admin/superuser/login \
  -H 'Content-Type: application/json' \
  -d '{"clientId":"admin","clientSecret":"secret"}'
```

## Backend TLS and mutual TLS

Without a `tls` block, Kerberos connects to a backend over plain `http://`. When a backend has a `tls` block, Kerberos connects with `https://host:port` and verifies the backend's certificate.

```json
{
  "gateway": {
    "router": {
      "backends": [
        {
          "name": "public-api",
          "host": "api.example.com",
          "port": 443,
          "tls": { "rootCAFile": "" }
        },
        {
          "name": "tls-echo",
          "host": "tls-echo",
          "port": 15000,
          "tls": { "rootCAFile": "/certs/ca.crt" }
        },
        {
          "name": "mtls-echo",
          "host": "mtls-echo",
          "port": 15000,
          "tls": {
            "rootCAFile": "/certs/ca.crt",
            "clientCertFile": "/certs/krb_client.pem",
            "clientKeyFile": "/certs/krb_client.key"
          }
        }
      ]
    }
  }
}
```

| Key | Type | Default | Description |
| --- | --- | --- | --- |
| `tls` | object | not set (plain HTTP) | When present, Kerberos uses HTTPS for this backend. |
| `tls.rootCAFile` | string | required when `tls` is present | Path to a PEM bundle of CA certificates used to verify the backend's certificate. If it is set to an empty string (`""`), the system CA pool is used. If a file is given, only the CAs in that file are trusted, not the system pool. |
| `tls.clientCertFile` | string | not set | Path to the PEM client certificate that Kerberos presents to the backend (mTLS). Must be set together with `clientKeyFile`. |
| `tls.clientKeyFile` | string | not set | Path to the PEM private key for `clientCertFile`. Must be set together with `clientCertFile`. |
| `tls.insecureSkipVerify` | boolean | `false` | Turns off verification of the backend's certificate. See the warning below. |

How verification works:

- The backend's certificate must be valid for the configured `host`. The certificate's Subject Alternative Names must include that DNS name, or that IP address if `host` is an IP. For example, the test certificate for the `tls-echo` backend has the SAN `DNS:tls-echo`.
- The certificate must chain to a CA in `rootCAFile`, or to a CA in the system pool when `rootCAFile` is `""`. The official Kerberos image includes the standard public CA certificates.
- For mTLS, the backend decides whether to accept Kerberos's client certificate. Kerberos only presents it.

> **Warning:** `insecureSkipVerify: true` accepts any certificate, including expired, self-signed or forged ones. The connection is still encrypted, but it is not authenticated, so a man-in-the-middle can read and change traffic. Use it only for local testing. `rootCAFile` is still required with `insecureSkipVerify`. Set it to `""` if you have no CA file.

If the TLS handshake with a backend fails, for example because the certificate is untrusted, the host name does not match, or the backend rejects the client certificate, the client gets the same response as for any other upstream failure: `500` with `{"errors":["failed to forward request"]}`. The real cause is in the Kerberos log. See [Routing](./routing.md).

## Certificate loading and startup errors

All certificate files are read once, at startup.

- **Backend files** (`rootCAFile`, `clientCertFile`, `clientKeyFile`) are loaded while Kerberos builds its backend connections. If a file is missing or unreadable, `rootCAFile` contains no valid PEM certificate, or the client certificate and key do not match, Kerberos logs the error and exits with code 1. The log message names the backend, for example `building transport for backend "mtls-echo": reading root CA bundle "/certs/ca.crt": ...` or `... no valid PEM certificates found in "/certs/ca.crt"`.
- **Server files** (`gateway.tls` and `admin.api.tls`) are loaded when each listener starts. If a file is missing, unreadable or invalid, Kerberos logs `GW server start error` or `Admin server start error` with the cause and exits with code 1.
- **Config validation** catches structural mistakes before any files are read. Examples are a `tls` block without `serverCertFile`/`serverKeyFile`, a backend `tls` block without `rootCAFile`, or only one of `clientCertFile`/`clientKeyFile`. See [Configuration](./configuration.md).

Because files are read only at startup, Kerberos does not notice renewed certificates. Restart it after you rotate certificates or CA bundles.

## Secure cookies need HTTPS

Kerberos always sets the `Secure` attribute on its session, refresh and CSRF cookies. Browsers store and send `Secure` cookies only over HTTPS. The exception is `http://localhost`, which most browsers treat as secure.

In practice:

- For browser-based login to work on any host other than `localhost`, browsers must reach the admin port over HTTPS. This applies to both the admin API and the basic-auth API. Browsers must also reach the gateway over HTTPS so that the session cookie is sent with gateway requests.
- The HTTPS can come from Kerberos itself (`gateway.tls`, `admin.api.tls`) or from a TLS-terminating proxy in front of it, because the browser only needs to see HTTPS.
- Non-browser clients such as `curl` or the Go clients send cookies however you tell them to, so plain HTTP works for them in testing.

See [Sessions](./sessions.md) for cookie names, lifetimes, `domain` and `sameSite`.

## Running behind a TLS-terminating proxy

Instead of configuring certificates in Kerberos, you can put a load balancer, ingress controller or reverse proxy in front of it to handle TLS. Kerberos then serves plain HTTP on its ports.

Things to keep in mind:

- The hop from the proxy to Kerberos is unencrypted. Keep it on a private network. If you need that hop encrypted, configure `gateway.tls`/`admin.api.tls` as well and have the proxy re-encrypt.
- Kerberos does not read `X-Forwarded-Proto` or similar headers. This does not matter for cookies, which are always marked `Secure`.
- Kerberos does not add `X-Forwarded-*` headers to backend requests. Headers the proxy adds are passed through to backends unchanged. See [Routing](./routing.md).
- Route both ports through the proxy if browsers use both: 30000 for the gateway and 30001 for the admin and login APIs.
- Use the TLS settings of the proxy to control TLS versions, cipher suites or client-certificate checks. Kerberos does not offer these settings (see [Limitations](#limitations-and-things-to-know)).

## Example: certificates mounted as files

This example runs Kerberos with HTTPS on both ports and two TLS backends, one of which uses mTLS. It is based on the repository's security test environment (`test/config/security.json`, `test/compose/security/compose.yaml`, certificates in `test/certs/`).

`krb.json`:

```json
{
  "admin": {
    "api": {
      "tls": {
        "serverCertFile": "/certs/krb_server.pem",
        "serverKeyFile": "/certs/krb_server.key"
      }
    }
  },
  "gateway": {
    "tls": {
      "serverCertFile": "/certs/krb_server.pem",
      "serverKeyFile": "/certs/krb_server.key"
    },
    "router": {
      "backends": [
        {
          "name": "tls-echo",
          "host": "tls-echo",
          "port": 15000,
          "tls": { "rootCAFile": "/certs/ca.crt" }
        },
        {
          "name": "mtls-echo",
          "host": "mtls-echo",
          "port": 15000,
          "tls": {
            "rootCAFile": "/certs/ca.crt",
            "clientCertFile": "/certs/krb_client.pem",
            "clientKeyFile": "/certs/krb_client.key"
          }
        }
      ]
    }
  }
}
```

`compose.yaml`:

```yaml
services:
  kerberos:
    image: ghcr.io/trebent/kerberos:latest
    command: --config /config/krb.json
    ports:
      - 30000:30000
      - 30001:30001
    environment:
      - LOG_TO_CONSOLE=1
    volumes:
      - ./krb.json:/config/krb.json:ro
      - ./certs:/certs:ro

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
      # Require a client certificate signed by this CA
      - TLS_CLIENT_CA_FILE=/certs/ca.crt
    volumes:
      - ./certs:/certs:ro
```

The image runs as a non-root user, so the mounted certificate and key files must be readable by that user. For example, run `chmod a+r` on the files in a test setup, or use your orchestrator's secret mechanism with appropriate permissions.

Test it:

```bash
curl --cacert certs/ca.crt https://localhost:30000/gw/backend/tls-echo/hello
curl --cacert certs/ca.crt https://localhost:30000/gw/backend/mtls-echo/hello
```

### Generating local certificates with mkcert

For local development, [mkcert](https://github.com/FiloSottile/mkcert) creates a local CA that your browser and system trust, and issues certificates from it. The repository's staging environment (`make compose/staging/up`, using `test/certs/staging/make_certs.sh` and `test/config/staging.json`) works this way:

```bash
mkcert -install
mkcert -cert-file certs/krb.pem -key-file certs/krb-key.pem localhost gw.example.test admin.example.test
# Client certificate for backend mTLS
mkcert -client -cert-file certs/krb-client.pem -key-file certs/krb-client-key.pem kerberos
# CA to use as rootCAFile, or for curl --cacert
cp "$(mkcert -CAROOT)/rootCA.pem" certs/rootCA.pem
```

Include every host name that clients and Kerberos will use in the certificate: the public names of Kerberos, and each backend's `host` value in the backends' certificates. Never share the mkcert CA key (`rootCA-key.pem`). Do not use mkcert certificates in production.

## Admin connector

The [admin connector](./admin-connector.md) is a separate binary with its own TLS settings:

| Key | Type | Description |
| --- | --- | --- |
| `tls.serverCertFile` | string | PEM server certificate for the connector's own listener. |
| `tls.serverKeyFile` | string | PEM private key for that certificate. |
| `targetTls.rootCAFile` | string | CA bundle used to verify the target's certificate (`""` = system pool). |
| `targetTls.insecureSkipVerify` | boolean | Turns off target certificate verification. |

The connector serves HTTPS only when both `tls.serverCertFile` and `tls.serverKeyFile` are non-empty. Otherwise it serves plain HTTP. The connector's configuration schema currently rejects any `targetTls` block. In practice, the connector reaches its target over plain HTTP. See [Admin connector](./admin-connector.md) for details.

## Limitations and things to know

- **No TLS tuning.** You cannot configure the minimum or maximum TLS version, cipher suites, curves or ALPN protocols. Kerberos uses the Go standard library defaults: TLS 1.2 and 1.3 with Go's default cipher suites.
- **No client-certificate authentication on the Kerberos ports.** The gateway and admin ports do not request or verify client certificates. mTLS is only available on the connection from Kerberos to backends. If you need client certificates at the edge, check them in a proxy in front of Kerberos.
- **HTTPS only, no redirect.** When server TLS is enabled on a port, plain HTTP on that port is rejected with `400`. Kerberos cannot serve HTTP and HTTPS on the same port, and it cannot redirect HTTP to HTTPS.
- **No HSTS.** Kerberos does not send `Strict-Transport-Security` or other security response headers. Add them at a proxy if you need them.
- **Certificates are read only at startup.** Restart Kerberos after rotating them.
- **One CA setting per backend.** A file in `rootCAFile` replaces the system pool for that backend. It does not add to it.
- **Backend connections use HTTP/1.1**, including over TLS.
- **The backend host name is used for verification.** If you address a backend by IP address or by an internal alias, its certificate must contain that exact name or IP.
- **TLS errors look like other upstream errors** (`500 failed to forward request`). Check the Kerberos log for the cause.
- **Database TLS is configured separately.** PostgreSQL connections use `persistence.postgres.sslMode`. See [Persistence](./persistence.md).

## Related docs

- [Configuration](./configuration.md): the config file and all sections
- [Deployment](./deployment.md): ports, health checks and the production security checklist
- [Routing](./routing.md): backends, forwarding and upstream errors
- [Sessions](./sessions.md): cookies and why they require HTTPS
- [Admin connector](./admin-connector.md): the connector's `tls` and `targetTls`
- [CORS](./cors.md): origins must match the scheme (`https://...`) that browsers use
