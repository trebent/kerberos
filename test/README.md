# Testing

This directory contains composables, configuration, certificates and more for testing Kerberos.

## Integration

The integration suite tests the core functionality of Kerberos meaning it's by far the biggest suite. It covers the gateway API, admin API, basic authentication and OAS as well as some corner cases.

## Security

The security suite tests TLS and some Origin restriction policies. It's broken out of the integration suite not to muddle the composable with another Kerberos instance.

## Connector

Specific connector tests.

## Staging

The staging environment does not have a test suite, it's intended as the standard deployment for Kerberos. Note that you are expected to set some `/etc/hosts` entries to use it as intended:

```bash
127.0.0.1 trebent.test
127.0.0.1 gw.trebent.test
127.0.0.1 admin.trebent.test
127.0.0.1 grafana.trebent.test
127.0.0.1 jaeger.trebent.test
```

To generate the certificates needed for a local staging environment, check `certs/staging/make_certs.sh`.
