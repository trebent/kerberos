//nolint:gochecknoglobals // Package env provides environment variable parsing for the Kerberos service.
package env

import (
	"github.com/trebent/envparser"
)

var (
	Version = envparser.Register(&envparser.Opts[string]{
		Name:  "VERSION",
		Desc:  "Version of the Kerberos service.",
		Value: "unset",
	})

	LogToConsole = envparser.Register(&envparser.Opts[bool]{
		Name: "LOG_TO_CONSOLE",
		Desc: "Set to log to console.",
	})
	LogVerbosity = envparser.Register(&envparser.Opts[int]{
		Name:     "LOG_VERBOSITY",
		Desc:     "Set the log verbosity.",
		Validate: validateGreaterThanOrEqualToZero,
	})

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
		Name:     "OTEL_EXPORTER_PROMETHEUS_PORT",
		Desc:     "OpenTelemetry Prometheus port.",
		Value:    9464, //nolint:mnd
		Create:   true,
		Validate: validatePort,
	})

	Port = envparser.Register(&envparser.Opts[int]{
		Name:     "PORT",
		Desc:     "Port for the Kerberos API GW server.",
		Value:    30000, // nolint: mnd
		Validate: validatePort,
	})
	ReadTimeoutSeconds = envparser.Register(&envparser.Opts[int]{
		Name:     "READ_TIMEOUT_SECONDS",
		Desc:     "Read timeout in seconds.",
		Value:    5, // nolint: mnd
		Validate: validateGreaterThanZero,
	})
	WriteTimeoutSeconds = envparser.Register(&envparser.Opts[int]{
		Name:     "WRITE_TIMEOUT_SECONDS",
		Desc:     "Write timeout in seconds.",
		Value:    5, // nolint: mnd
		Validate: validateGreaterThanZero,
	})

	RouteJSONFile = envparser.Register(&envparser.Opts[string]{
		Name:     "ROUTE_JSON_FILE",
		Desc:     "JSON file to load routes from.",
		Value:    "./routes.json",
		Validate: validateFilePath,
	})
	ObsJSONFile = envparser.Register(&envparser.Opts[string]{
		Name:     "OBS_JSON_FILE",
		Desc:     "JSON file to load observability settings from.",
		Value:    "",
		Validate: validateFilePath,
	})
	AuthJSONFile = envparser.Register(&envparser.Opts[string]{
		Name:     "AUTH_JSON_FILE",
		Desc:     "JSON file to load authentication settings from.",
		Value:    "",
		Validate: validateFilePath,
	})
	OASJSONFile = envparser.Register(&envparser.Opts[string]{
		Name:     "OAS_JSON_FILE",
		Desc:     "JSON file to load OAS settings from.",
		Value:    "",
		Validate: validateFilePath,
	})

	DBDirectory = envparser.Register(&envparser.Opts[string]{
		Name:     "DB_DIRECTORY",
		Desc:     "Path to the directory where DB files will be stored.",
		Value:    "",
		Validate: validateDirPath,
	})
)

func Parse() error {
	return envparser.Parse()
}
