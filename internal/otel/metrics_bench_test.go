package otel

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/trebent/kerberos/internal/response"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
)

func BenchmarkBump(b *testing.B) {
	provider := sdkmetric.NewMeterProvider(sdkmetric.WithReader(sdkmetric.NewManualReader()))
	b.Cleanup(func() { _ = provider.Shutdown(context.Background()) })
	otel.SetMeterProvider(provider)

	metrics, err := StandardHTTPMetrics("", "1.0.0")
	if err != nil {
		b.Fatal(err)
	}

	wrapper := response.NewResponseWrapper(httptest.NewRecorder())
	wrapper.WriteHeader(http.StatusOK)
	req := httptest.NewRequest(http.MethodGet, "http://example.com/gw/backend/echo/hello", nil)
	backend := attribute.String("krb.backend", "echo")
	ctx := context.Background()

	b.ReportAllocs()
	for b.Loop() {
		metrics.Bump(ctx, wrapper, nil, req, time.Millisecond, backend)
	}
}
