# Routing

HTTP handler call order:

1. OTEL
2. Router (fetch backend)
3. Forward (using router backend)

## Routing

Kerberos will forward requests following this format:

URL: `/gw/backend/<backend-name>/<backend-path>`

The router will extract the `<backend-name>` and lookup if such a backend has been registered with Kerberos. If one is found, the request is forwarded to the registered backend's URL with the `<backend-path>` appended.
