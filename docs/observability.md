# Observability

Kerberos emits OpenTelemetry metrics and traces around gateway request handling.

## Metrics

Kerberos request/response metrics include:

- `request_count_total`
- `request_duration_milliseconds_bucket`
- `request_size_bytes_bucket`
- `response_size_bytes_bucket`
- `response_total`

Histogram-style metrics also expose `_sum` and `_count` companions.

### Metric labels

Common labels:

- `krb_backend`
- `http_method`

Additional `response_total` label:

- `http_status_code`

## Tracing

Kerberos starts a server span when a request enters the gateway flow.

High-level tracing behavior:

- Incoming trace headers are extracted and used as parent context when present.
- A new span is created for gateway handling.
- The resulting trace context is propagated to forwarded backend requests.
- This allows backend spans to appear under the same distributed trace.

## Flow relationship

Observability wraps the full routing/forwarding lifecycle, so metrics and trace completion represent end-to-end gateway processing (including early rejections from routing, auth, or validation).

## Runtime notes

- When observability is disabled in config, Kerberos still preserves required flow context behavior; full OTEL metrics/tracing export is what is disabled.
- Debug sessions are separate from tracing and can be used together to inspect gateway decisions in detail.
