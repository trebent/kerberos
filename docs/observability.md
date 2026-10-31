# Observability

Kerberos produces OpenTelemetry metrics and traces for every request that passes through the gateway or the admin API, and writes a log line per request. Use them to see traffic and error rates per backend, to follow a request from the client through Kerberos into your backend in a tracing tool such as Jaeger, and to build dashboards and alerts.

Exporters (where metrics and traces are sent) are configured with the standard OpenTelemetry environment variables. The Kerberos configuration file only switches the instrumentation on or off.

## Overview

| Signal | What you get | Configured by |
| --- | --- | --- |
| Metrics | Request count, response count by status code, request/response size and request duration, labelled by backend and HTTP method. Separate `admin_` metrics for the admin API. Go runtime metrics. | `observability` config section, `OTEL_METRICS_EXPORTER` and related env vars |
| Traces | One server span per request on the gateway and on the admin API. W3C trace context is read from incoming requests and passed on to backends. | `observability` config section, `OTEL_TRACES_EXPORTER`, `OTEL_TRACES_SAMPLER` and related env vars |
| Logs | Structured (JSON) or human-readable logs on stdout, with one summary line per request. | `LOG_TO_CONSOLE`, `LOG_VERBOSITY` |

For inspecting what happened to one specific request inside the gateway (which component accepted or rejected it, and why), use [debug sessions](./admin-debugging.md) instead. They work independently of the settings on this page.

## Configuration

The `observability` section of the configuration file is optional.

```json
{
  "observability": {
    "enabled": true,
    "runtimeMetrics": true
  }
}
```

| Key | Type | Default | Description |
| --- | --- | --- | --- |
| `enabled` | boolean | `true` | Turns metrics and tracing on or off for the whole process (gateway and admin API). |
| `runtimeMetrics` | boolean | `true` | Also export Go runtime metrics (memory, goroutines, GC goal). Has no effect when `enabled` is `false`. |

Leaving the section out, or writing `"observability": {}`, gives you both defaults (`true`). The schema is [`internal/config/schemas/observability_schema.json`](../internal/config/schemas/observability_schema.json).

### What `enabled: false` changes

When observability is disabled:

- No metrics or traces are recorded and no exporter is started. `OTEL_*` environment variables have no effect, and the Prometheus metrics endpoint stays closed.
- Trace headers sent by the client are forwarded to the backend unchanged, like any other request header.
- The observability component (`obs`) no longer appears in the flow returned by the admin flow inspection endpoint (see [Admin API](./admin-api.md)).

It does not turn off:

- Request logging. Each gateway request is still logged, and the log format changes slightly (see [Logging](#logging)).
- [Debug sessions](./admin-debugging.md). These keep working; recorded requests have no `obs` steps.
- Routing, authentication, request validation and forwarding, which behave exactly the same.

To keep the instrumentation active but stop sending data anywhere, leave `enabled` on and set `OTEL_TRACES_EXPORTER=none` and/or `OTEL_METRICS_EXPORTER=none` instead.

## Exporters and environment variables

Kerberos uses the OpenTelemetry Go SDK's automatic exporter selection, so the usual `OTEL_*` variables apply. The table lists the ones you are most likely to need; for the full list see the [OpenTelemetry SDK environment variable specification](https://opentelemetry.io/docs/specs/otel/configuration/sdk-environment-variables/). All Kerberos-specific environment variables are described in [Deployment](./deployment.md).

| Variable | Default | Values and notes |
| --- | --- | --- |
| `OTEL_TRACES_EXPORTER` | `otlp` | `otlp`, `console` (print spans to stdout), `none`. |
| `OTEL_METRICS_EXPORTER` | `otlp` | `otlp`, `prometheus` (serve a scrape endpoint), `console`, `none`. |
| `OTEL_EXPORTER_OTLP_ENDPOINT` | `http://localhost:4318` for `http/protobuf`, `http://localhost:4317` for `grpc` | Base URL of your OTLP receiver (collector, Jaeger, Tempo, ...). Use the signal-specific `OTEL_EXPORTER_OTLP_TRACES_ENDPOINT` / `OTEL_EXPORTER_OTLP_METRICS_ENDPOINT` to send traces and metrics to different places. With gRPC, an `http://` URL means a plaintext connection. |
| `OTEL_EXPORTER_OTLP_PROTOCOL` | `http/protobuf` | `http/protobuf` or `grpc`. Applies to all signals. |
| `OTEL_EXPORTER_OTLP_TRACES_PROTOCOL`, `OTEL_EXPORTER_OTLP_METRICS_PROTOCOL` | value of `OTEL_EXPORTER_OTLP_PROTOCOL` | Per-signal override of the protocol. |
| `OTEL_EXPORTER_PROMETHEUS_HOST` | `localhost` | Interface the Prometheus endpoint binds to (only with `OTEL_METRICS_EXPORTER=prometheus`). See the note below for containers. |
| `OTEL_EXPORTER_PROMETHEUS_PORT` | `9464` | Port of the Prometheus endpoint. Metrics are served at `/metrics`. |
| `OTEL_TRACES_SAMPLER` | `parentbased_always_on` | `always_on`, `always_off`, `traceidratio`, `parentbased_always_on`, `parentbased_always_off`, `parentbased_traceidratio`. |
| `OTEL_TRACES_SAMPLER_ARG` | - | Sampling ratio between `0` and `1` for the `*traceidratio` samplers, for example `0.1`. |
| `OTEL_RESOURCE_ATTRIBUTES` | - | Extra resource attributes, for example `deployment.environment=prod`. `service.name` and `service.version` are set by Kerberos (see [Tracing](#tracing)). |

Things to know:

- **Both exporters default to OTLP.** If you set nothing, Kerberos sends traces and metrics to `localhost:4318`. Point the exporters at your receiver, or set them to `none`.
- **Prometheus endpoint in containers.** In a container, set `OTEL_EXPORTER_PROMETHEUS_HOST=0.0.0.0`, or the container's own service name as the example below does, so that Prometheus can reach the endpoint. The metrics port is separate from the gateway and admin ports; publish it if Prometheus runs outside the container network.
- **Use a free port and valid values.** Kerberos checks the Prometheus address and the `OTEL_*` exporter values at startup, and logs `Failed to instrument OpenTelemetry` if one is wrong.
- **Sampling in production.** By default every trace is kept, unless the caller has already decided not to sample it. That suits development. In production, keep a share of traces with a ratio sampler that still follows the caller's decision:

  ```sh
  OTEL_TRACES_SAMPLER=parentbased_traceidratio
  OTEL_TRACES_SAMPLER_ARG=0.1   # keep 10 % of new traces
  ```

  Metrics are not affected by sampling.

### Example: Prometheus scrape and OTLP traces to Jaeger

This is the setup used by the repository's integration environment (`test/compose/integration/compose.yaml`) and by the Compose file that [krbctl](./krbctl.md) generates with `--obs-stack`. Metrics are scraped by Prometheus; traces go to Jaeger over OTLP/gRPC.

```yaml
services:
  kerberos:
    image: "ghcr.io/trebent/kerberos:latest"
    command: --config /krb.json
    ports:
      - 30000:30000
      - 30001:30001
      - 9464:9464          # only needed if Prometheus runs outside this Compose network
    environment:
      - OTEL_EXPORTER_OTLP_ENDPOINT=http://jaeger:4317
      - OTEL_EXPORTER_OTLP_TRACES_PROTOCOL=grpc
      - OTEL_METRICS_EXPORTER=prometheus
      - OTEL_EXPORTER_PROMETHEUS_HOST=kerberos   # or 0.0.0.0
      - OTEL_EXPORTER_PROMETHEUS_PORT=9464
      # Recommended in production:
      # - OTEL_TRACES_SAMPLER=parentbased_traceidratio
      # - OTEL_TRACES_SAMPLER_ARG=0.1
    volumes:
      - ./krb.json:/krb.json:ro

  jaeger:
    image: "jaegertracing/jaeger:2.20.0"
    ports:
      - 16686:16686        # Jaeger UI

  prometheus:
    image: "prom/prometheus:v3"
    command: ["--config.file=/prometheus.yml"]
    ports:
      - 9090:9090
    volumes:
      - ./prometheus.yml:/prometheus.yml
```

`prometheus.yml`:

```yaml
global:
  scrape_interval: 15s

scrape_configs:
  - job_name: kerberos
    static_configs:
      - targets: ["kerberos:9464"]
```

Because `OTEL_METRICS_EXPORTER=prometheus` is set, metrics are not pushed over OTLP; only traces go to `jaeger:4317`.

The same approach works for the [echo test backend](./echo.md) and [admin-connector](./admin-connector.md), which read the same `OTEL_*` variables. If several of these processes run on one host, give each its own `OTEL_EXPORTER_PROMETHEUS_PORT`.

## Metrics

### Gateway metrics

Recorded for every request handled by the gateway under `/gw/` (not for `/gw/health`). Names are as they appear in Prometheus; over OTLP the instrument names are `request.count`, `request.size`, `request.duration`, `response` and `response.size`.

| Prometheus name | Type | Unit | Labels | Description |
| --- | --- | --- | --- | --- |
| `request_count_total` | counter | - | `krb_backend`, `http_method` | Number of requests. |
| `response_total` | counter | - | `krb_backend`, `http_method`, `http_status_code` | Number of responses, by status code. |
| `request_duration_milliseconds` | histogram | ms | `krb_backend`, `http_method` | Time spent handling the request: routing, authentication, validation, forwarding to the backend and relaying the response. Buckets: 1, 10, 100, 1000, 10000. |
| `request_size_bytes` | histogram | bytes | `krb_backend`, `http_method` | Request body bytes read by Kerberos. Buckets: 0, 100, 1e3, 1e4, 1e5, 1e6, 1e7, 1e8. |
| `response_size_bytes` | histogram | bytes | `krb_backend`, `http_method` | Response body bytes sent to the client. Same buckets as `request_size_bytes`. |

Histograms appear in Prometheus as `_bucket`, `_sum` and `_count` series (for example `request_duration_milliseconds_bucket`). The Prometheus exporter also adds `otel_scope_name` (and related `otel_scope_*`) labels to each series and a `target_info` series carrying the resource attributes (`service_name="krb"`, `service_version`).

Notes:

- **`krb_backend="unknown"`.** The backend label is filled in once Kerberos has matched the request to a configured backend. These responses are recorded with `krb_backend="unknown"`:
  - `400` when the path is not of the form `/gw/backend/<name>/...` (for example a missing trailing slash);
  - `404` when no backend with that name exists;
  - `403` when the request's `Origin` is not allowed for the backend (see [CORS](./cors.md));
  - `204` answers to CORS preflight (`OPTIONS`) requests, even though the backend is known.

  Requests rejected later, by authentication or request validation, carry the real backend name.
- **Duration is in whole milliseconds.** A request that takes 15.7 ms is recorded as `15`. The histogram is built for spotting slow requests.
- **Request size counts the bytes read.** A request that is rejected before its body is forwarded (for example by authentication) records the bytes read up to that point.
- `request_count_total` and `response_total` are recorded together, so the sum of `response_total` over status codes equals `request_count_total`.

### Admin API metrics

The admin server (`ADMIN_PORT`, including the basic-auth API under `/api/auth/basic/`) records the same five metrics with an `admin_` prefix:

| Prometheus name | Type | Labels |
| --- | --- | --- |
| `admin_request_count_total` | counter | `http_method` |
| `admin_response_total` | counter | `http_method`, `http_status_code` |
| `admin_request_duration_milliseconds` | histogram (1, 10, 100, 1000, 10000 ms) | `http_method` |
| `admin_request_size_bytes` | histogram | `http_method` |
| `admin_response_size_bytes` | histogram | `http_method` |

Admin metrics are labelled by HTTP method and status code.

### Runtime metrics

With `runtimeMetrics: true`, the Go runtime metrics from OpenTelemetry's runtime instrumentation are exported as well, including:

| OpenTelemetry name | Prometheus name |
| --- | --- |
| `go.memory.used` | `go_memory_used_bytes` |
| `go.memory.limit` | `go_memory_limit_bytes` |
| `go.memory.allocated` | `go_memory_allocated_bytes_total` |
| `go.memory.allocations` | `go_memory_allocations_total` |
| `go.memory.gc.goal` | `go_memory_gc_goal_bytes` |
| `go.goroutine.count` | `go_goroutine_count` |
| `go.processor.limit` | `go_processor_limit` |
| `go.config.gogc` | `go_config_gogc_percent` |

### Example PromQL queries

Request rate per backend:

```promql
sum by (krb_backend) (rate(request_count_total[5m]))
```

Share of server errors (5xx) per backend:

```promql
sum by (krb_backend) (rate(response_total{http_status_code=~"5.."}[5m]))
/
sum by (krb_backend) (rate(response_total[5m]))
```

Replace `5..` with `4..|5..` to include client errors such as `401` and `403` from authentication.

95th percentile latency per backend, in milliseconds:

```promql
histogram_quantile(
  0.95,
  sum by (krb_backend, le) (rate(request_duration_milliseconds_bucket[5m]))
)
```

With buckets at 1, 10, 100, 1000 and 10000 ms, read the result as a rough size rather than an exact value.

Exclude unmatched traffic from any of these with `krb_backend!="unknown"`.

## Tracing

Every trace exported by Kerberos carries these resource attributes:

| Attribute | Value |
| --- | --- |
| `service.name` | `krb` (fixed) |
| `service.version` | value of the `VERSION` environment variable (see [Deployment](./deployment.md#environment-variables); the published images set it to the release version) |

### Gateway spans

For each request under `/gw/`, Kerberos creates one span of kind `SERVER`:

| Property | Value |
| --- | --- |
| Instrumentation scope | `krb` |
| Span name | `<METHOD> <request URI>`, for example `GET /gw/backend/echo/hello?x=1` |
| `http.method` | request method |
| `http.url` | request URI as received, including the `/gw/backend/<name>/` prefix and the query string |
| `krb.backend` | backend name, or `unknown` in the same cases as the metric label |
| Status | `Error` with the HTTP status text (for example `Unauthorized`) when the response status is 400 or higher, otherwise `Ok` |

The one server span covers routing, authentication, validation and the backend call. Use [debug sessions](./admin-debugging.md) for a step-by-step breakdown.

The span name and `http.url` contain the full path and query string, so keep secrets out of query parameters. If your paths contain IDs, expect many different span names.

### Context propagation

Kerberos uses the W3C [Trace Context](https://www.w3.org/TR/trace-context/) (`traceparent`, `tracestate`) and [Baggage](https://www.w3.org/TR/baggage/) (`baggage`) headers:

- **Incoming:** if the client sends a `traceparent` header, the Kerberos span becomes its child, and with a `parentbased_*` sampler the client's sampling decision is kept.
- **Outgoing:** when forwarding to a backend, Kerberos writes `traceparent` (and `baggage`, if present) pointing at its own span. A backend that uses OpenTelemetry therefore shows up under the Kerberos span in the same trace.

### Admin spans

Requests to the admin server get the same kind of `SERVER` span, with instrumentation scope `krb-admin`, name `<METHOD> <request URI>`, the `http.method` and `http.url` attributes and the same error status rule. Incoming `traceparent` headers are followed here too.

## Logging

Kerberos logs to stdout. Two environment variables control the output (their defaults are listed in [Deployment: environment variables](./deployment.md#environment-variables)):

| Variable | Description |
| --- | --- |
| `LOG_TO_CONSOLE` | `false`: one JSON object per line (fields include `level`, `time`, `caller`, `name`, `msg`), suited for log collectors. `true`: coloured, human-readable lines for local development. |
| `LOG_VERBOSITY` | Amount of detail; must be `0` or higher. Higher values include everything from lower ones. |

Verbosity levels in use:

| Level | What is added |
| --- | --- |
| `0` | Startup and shutdown messages, errors, and per-request messages (see below). |
| `10` | Successful basic-auth logins (with username) and basic-auth API authorisation failures. |
| `20` | Decisions inside each step: sessions and cookies found or missing, CSRF and CORS checks, which auth method is used, OpenAPI checks, debug-session matching, forwarding. Useful for troubleshooting a route. |
| `50` | Authentication and authorisation steps for each gateway request, and completion of each backend call. |
| `100` | Very verbose: every response header copied from the backend, each response write, and configuration parsing details. |

> Levels above `0` are for troubleshooting. Keep production at `0`.

### Per-request log lines

Every gateway request produces a summary line at level `info` once the response has been sent:

```
GET /gw/backend/echo/hello 200
```

The message is `<METHOD> <original path> <status>`. Each line carries an `http.path` field with the original path. At verbosity `0` you will also see short messages from the components the request passed through, such as `Routing request`, `Authorizing request` (when [authentication](./authentication.md) is configured) and `Running OAS validation` (when [request validation](./request-validation.md) is configured).

When `observability.enabled` is `false`, the gateway logs `<METHOD> <path>` when the request arrives and `<METHOD> <path> <status>` when it completes, with the fields `path`, `method` and `http.status_code`.

The admin server logs the same `<METHOD> <path> <status>` line for each admin API request, whether or not observability is enabled.

`/gw/health` requests are not logged.

## Dashboards and Prometheus configuration from krbctl

When you scaffold a deployment with [krbctl](./krbctl.md) and choose the observability stack (`--obs-stack`), it generates, next to the Compose file and `krb.json`:

| File | Contents |
| --- | --- |
| `prometheus.yml` | Scrape interval 15 s and one job per selected scrape target: `kerberos` at `kerberos:9464`, `jaeger` at `jaeger:8888`, each selected backend at `<backend host>:9464`, and `connector:9464` when admin-connector is included. Choose targets with `--scrape-target` (by default: Kerberos, Jaeger and all backends). |
| `grafana/grafana-datasources.yml` | A Prometheus data source at `http://prometheus:9090`. |
| `grafana/grafana-dashboards.yml` and the dashboard JSON files | Three provisioned dashboards (see below). |
| `grafana/grafana.ini` | Grafana database settings and anonymous viewer access (`--grafana-anonymous`, on by default). |
| `jaeger.yml`, `jaeger-config-ui.json` | Jaeger configuration, receiving OTLP on port 4317. |

The provisioned Grafana dashboards are:

- **Kerberos HTTP** (`kerberos_http.json`): gateway request rate and request count per backend and method, responses per status code, and request duration, request size and response size buckets per HTTP method.
- **Kerberos runtime** (`kerberos_runtime.json`): goroutines, processor limit, memory used and limit, heap allocations, GC goal, and Prometheus exporter errors (`promhttp_metric_handler_errors_total`).
- **Prometheus status** (`prometheus.json`): scrape duration, samples and series per target, and `target_info`.

Grafana is published on port 3000. Open the Jaeger UI directly (port 16686, or through [admin-connector](./admin-connector.md) when you included it). The generated `krb.json` always contains `"observability": {"enabled": true, "runtimeMetrics": true}`; without `--obs-stack` the Compose file sets `OTEL_METRICS_EXPORTER=none` and `OTEL_TRACES_EXPORTER=none` instead.

The dashboard JSON files can also be imported into an existing Grafana. They expect a Prometheus data source with uid `prometheus`.

## Related docs

- [Debug sessions](./admin-debugging.md): capture per-request flow transitions for one backend.
- [Deployment](./deployment.md): all environment variables, ports and production checklist.
- [Configuration](./configuration.md): the configuration file and its other sections.
- [krbctl](./krbctl.md): generating Compose files, `krb.json` and the observability stack.
- [admin-connector](./admin-connector.md): its own metrics and how it exposes the Jaeger UI.
