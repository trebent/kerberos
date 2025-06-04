package env

import (
	"fmt"
	"os"

	"github.com/trebent/envparser"
)

// nolint: gochecknoglobals
var (
	Port = envparser.Register(&envparser.Opts[int]{
		Name:  "PORT",
		Desc:  "Port for the Kerberos API GW server.",
		Value: 30000, // nolint: mnd
		Validate: func(v int) error {
			if v < 1000 || v > 65535 {
				return fmt.Errorf("must be between 1000 and 65535: %d", v)
			}
			return nil
		},
	})
	TestEndpoint = envparser.Register(&envparser.Opts[bool]{
		Name:  "TEST_ENDPOINT",
		Desc:  "Set to enable a testing endpoint that can be used to test how Kerberos generates metrics for various requests.",
		Value: false,
	})
	LogToConsole = envparser.Register(&envparser.Opts[bool]{
		Name: "LOG_TO_CONSOLE",
		Desc: "Set to log to console.",
	})
	LogVerbosity = envparser.Register(&envparser.Opts[int]{
		Name: "LOG_VERBOSITY",
		Desc: "Set the log verbosity.",
		Validate: func(v int) error {
			if v < 0 {
				return fmt.Errorf("must be greater than or equal to 0: %d", v)
			}
			return nil
		},
	})
	ReadTimeoutSeconds = envparser.Register(&envparser.Opts[int]{
		Name:  "READ_TIMEOUT_SECONDS",
		Desc:  "Read timeout in seconds.",
		Value: 5, // nolint: mnd
		Validate: func(v int) error {
			if v < 1 {
				return fmt.Errorf("must be greater than 0: %d", v)
			}
			return nil
		},
	})
	WriteTimeoutSeconds = envparser.Register(&envparser.Opts[int]{
		Name:  "WRITE_TIMEOUT_SECONDS",
		Desc:  "Write timeout in seconds.",
		Value: 5, // nolint: mnd
		Validate: func(v int) error {
			if v < 1 {
				return fmt.Errorf("must be greater than 0: %d", v)
			}
			return nil
		},
	})
	RouteJSONFile = envparser.Register(&envparser.Opts[string]{
		Name:  "ROUTE_JSON_FILE",
		Desc:  "JSON file to load routes from.",
		Value: "./routes.json", // nolint: mnd
		Validate: func(path string) error {
			f, err := os.Open(path)
			if err != nil {
				return err
			}
			defer f.Close()
			return nil
		},
	})
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

func Parse() error {
	return envparser.Parse()
}
