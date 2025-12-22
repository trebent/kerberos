package env

import (
	"fmt"

	"github.com/trebent/envparser"
)

var (
	// OTEL conf.
	OtelMetricsExporter = envparser.Register(&envparser.Opts[string]{
		Name:           "OTEL_METRICS_EXPORTER",
		Desc:           "OpenTelemetry metrics exporter.",
		Value:          "none",
		Create:         true,
		AcceptedValues: []string{"none", "prometheus", "otlp", "console"},
	})
	OtelTracesExporter = envparser.Register(&envparser.Opts[string]{
		Name:           "OTEL_TRACES_EXPORTER",
		Desc:           "OpenTelemetry traces exporter.",
		Value:          "none",
		Create:         true,
		AcceptedValues: []string{"none", "otlp", "console"},
	})
	OtelTracesExporterProtocol = envparser.Register(&envparser.Opts[string]{
		Name:           "OTEL_EXPORTER_OTLP_TRACES_PROTOCOL",
		Desc:           "OpenTelemetry traces exporter protocol.",
		Value:          "grpc",
		Create:         true,
		AcceptedValues: []string{"http/protobuf", "grpc"},
	})
	OtelExporterOTLPEndpoint = envparser.Register(&envparser.Opts[string]{
		Name: "OTEL_EXPORTER_OTLP_ENDPOINT",
		Desc: "OpenTelemetry OTLP exporter endpoint.",
		// This ensures plain HTTP can be used for OTEL type exports.
		Value:  "http://localhost:4317",
		Create: true,
	})
	OtelExporterPrometheusHost = envparser.Register(&envparser.Opts[string]{
		Name:   "OTEL_EXPORTER_PROMETHEUS_HOST",
		Desc:   "OpenTelemetry Prometheus host.",
		Value:  "localhost",
		Create: true,
	})
	OtelExporterPrometheusPort = envparser.Register(&envparser.Opts[int]{
		Name:   "OTEL_EXPORTER_PROMETHEUS_PORT",
		Desc:   "OpenTelemetry Prometheus port.",
		Value:  9464, //nolint:mnd
		Create: true,
		Validate: func(v int) error {
			if v < 1000 || v > 65535 {
				return fmt.Errorf("must be between 1000 and 65535: %d", v)
			}
			return nil
		},
	})
)
