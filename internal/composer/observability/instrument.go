package obs

import (
	"context"

	"github.com/trebent/kerberos/internal/config"
	"github.com/trebent/kerberos/internal/otel"
	"github.com/trebent/zerologr"
)

// Instrument bootstraps the OpenTelemetry pipeline.
// If it does not return an error, make sure to call shutdown for proper cleanup.
func Instrument(
	ctx context.Context,
	cfg config.Map,
	serviceName,
	serviceVersion string,
) (func(context.Context) error, error) {
	obsCfg := config.AccessAs[*obsConfig](cfg, configName)
	if !obsCfg.Enabled || !obsCfg.RuntimeMetrics {
		zerologr.Info("Observability runtime metrics are disabled")
		return func(context.Context) error { return nil }, nil
	}

	return otel.Instrument(ctx, serviceName, serviceVersion)
}
