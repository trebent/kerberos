# Testing

This directory contains composables, configuration, certificates and more for testing Kerberos.

## Integration

The integration suite tests the core functionality of Kerberos meaning it's by far the biggest suite. It covers the gateway API, admin API, basic authentication and OAS as well as some corner cases.

## Security

The security suite tests TLS and some Origin restriction policies. It's broken out of the integration suite not to muddle the composable with another Kerberos instance.

## Connector

Specific connector tests.

## Staging

The staging environment does not have a test suite, it's intended as the standard deployment for Kerberos. Note that you are expected to set some `/etc/hosts` entries to use it as it's intended:

```bash
127.0.0.1 trebent.test
127.0.0.1 gw.trebent.test
127.0.0.1 admin.trebent.test
127.0.0.1 grafana.trebent.test
127.0.0.1 jaeger.trebent.test
```

### Certificates

To generate the certificates needed for a local staging environment, check `certs/staging/make_certs.sh`, this is also done automatically on `make compose/staging/up`. You need to have `mkcert` installed. This allows the staging environment to closely replicate *real* Kerberos deployments with actual certificate validation and HTTPs browser usage.

The staging environment certificates target dedicated hostnames; the Jaeger server certificate uses `jaeger`, grafana uses `grafana.trebent.test`, and the admin connector proxying Jaeger uses `jaeger.trebent.test`. Kerberos uses `gw.trebent.test` and `admin.trebent.test`, all certificates also use `localhost`.

### Admin cookie configuration

The staging configuration is set so that the `trebent.test` is allowed to use admin API cookies, so it's not mandatory to serve the Kerberos frontend from `admin.trebent.test`, for example. `admin.api.cookies.domain` is set to `trebent.test` for this purpose. This also allows you to use the connector served from `jaeger.trebent.test` as cookies are shared to subdomains.

### CORS

The Admin API is set up to whitelist only 1 CORS origin `https://trebent.test:4200` as it's assumed the staging is run together with `kerberos-fe` and the `run/staging` make-target. This should closely mimic a real deployment, where only 1 server serves the Kerberos frontend, if there is one. Basic auth allows all origins, assuming multi-tenant scenario with more than 1 organisation. Backend #1 allows all as well, while backend #2 completely dissallows CORS.
