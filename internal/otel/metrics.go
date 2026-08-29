package otel

import (
	"context"
	"io"
	"net/http"
	"time"

	"github.com/trebent/kerberos/internal/response"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
	semconv "go.opentelemetry.io/otel/semconv/v1.20.0"
)

type StdHTTPMetrics struct {
	requestCountCounter      metric.Int64Counter
	requestSizeHistogram     metric.Int64Histogram
	requestDurationHistogram metric.Float64Histogram

	responseCounter       metric.Int64Counter
	responseSizeHistogram metric.Int64Histogram
}

const (
	requestCounterName           = "request.count"
	requestSizeHistogramName     = "request.size"
	requestDurationHistogramName = "request.duration"

	responseCounterName       = "response"
	responseSizeHistogramName = "response.size"
)

func StandardHTTPMetrics(prefix, version string) (*StdHTTPMetrics, error) {
	var err error

	meter := otel.GetMeterProvider().Meter(
		"github.com/trebent/kerberos",
		metric.WithInstrumentationVersion(version),
	)

	metrics := &StdHTTPMetrics{}

	metrics.requestCountCounter, err = meter.Int64Counter(
		prefix+requestCounterName,
		metric.WithDescription("Measures the number of HTTP requests."),
	)
	if err != nil {
		return nil, err
	}

	metrics.requestSizeHistogram, err = meter.Int64Histogram(
		prefix+requestSizeHistogramName,
		metric.WithUnit("By"),
		metric.WithDescription("Measures the size of HTTP request bodies."),
		metric.WithExplicitBucketBoundaries(
			0,
			100,
			1000,
			10000,
			100000,
			1000000,
			10000000,
			100000000,
		),
	)
	if err != nil {
		return nil, err
	}

	metrics.requestDurationHistogram, err = meter.Float64Histogram(
		prefix+requestDurationHistogramName,
		metric.WithUnit("ms"),
		metric.WithDescription("Measures the time spent handling HTTP requests."),
		metric.WithExplicitBucketBoundaries(1, 10, 100, 1000, 10000),
	)
	if err != nil {
		return nil, err
	}

	metrics.responseCounter, err = meter.Int64Counter(
		prefix+responseCounterName,
		metric.WithDescription("Keeps track of HTTP response status code counts."),
	)
	if err != nil {
		return nil, err
	}

	metrics.responseSizeHistogram, err = meter.Int64Histogram(
		prefix+responseSizeHistogramName,
		metric.WithUnit("By"),
		metric.WithDescription("Measures the size of HTTP response bodies."),
		metric.WithExplicitBucketBoundaries(
			0,
			100,
			1000,
			10000,
			100000,
			1000000,
			10000000,
			100000000,
		),
	)
	if err != nil {
		return nil, err
	}

	return metrics, nil
}

func (s *StdHTTPMetrics) Bump(
	ctx context.Context,
	responseWrapper http.ResponseWriter,
	bodyWrapper io.ReadCloser,
	req *http.Request,
	duration time.Duration,
	attributes ...attribute.KeyValue,
) {
	//nolint:errcheck // no point
	statusCodeOpt := metric.WithAttributes(
		semconv.HTTPStatusCode(responseWrapper.(*response.Wrapper).StatusCode()),
	)
	requestMeta := metric.WithAttributes(semconv.HTTPMethod(req.Method))
	krbMetricMeta := metric.WithAttributes(attributes...)

	// Request
	var reqBytes int64
	if bodyWrapper != nil && bodyWrapper != http.NoBody {
		//nolint:errcheck // no point
		reqBytes = bodyWrapper.(*response.BodyWrapper).NumBytes()
	}

	s.requestCountCounter.Add(ctx, 1, requestMeta, krbMetricMeta)
	s.requestSizeHistogram.Record(ctx, reqBytes, requestMeta, krbMetricMeta)
	s.requestDurationHistogram.Record(
		ctx,
		float64(duration/time.Millisecond),
		requestMeta,
		krbMetricMeta,
	)

	// Response
	s.responseCounter.Add(ctx, 1, statusCodeOpt, requestMeta, krbMetricMeta)
	//nolint:errcheck // no point
	s.responseSizeHistogram.Record(
		ctx, responseWrapper.(*response.Wrapper).NumBytes(), requestMeta, krbMetricMeta,
	)
}
