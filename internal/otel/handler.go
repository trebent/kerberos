package otel

import (
	"net/http"
	"time"

	"github.com/go-logr/logr"
	"github.com/trebent/kerberos/internal/version"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/propagation"
	semconv "go.opentelemetry.io/otel/semconv/v1.20.0"
	"go.opentelemetry.io/otel/trace"
)

type obs struct {
	spanOpts                 []trace.SpanStartOption
	requestCountCounter      metric.Int64Counter
	requestSizeHistogram     metric.Int64Histogram
	requestDurationHistogram metric.Float64Histogram
	responseCounter          metric.Int64Counter
	responseSizeHistogram    metric.Int64Histogram
}

const (
	tracerName = "krb"
	spanName   = "krb"

	requestCountCounterName      = "request.count"
	requestSizeHistogramName     = "request.size"
	requestDurationHistogramName = "request.duration"

	responseCounterName       = "response"
	responseSizeHistogramName = "response.size"
)

var meter = otel.GetMeterProvider().Meter(
	"github.com/trebent/kerberos/forwarder",
	metric.WithInstrumentationVersion(version.Version()),
)

func Middleware(next http.Handler, logger logr.Logger) http.Handler {
	o := &obs{
		spanOpts: []trace.SpanStartOption{
			trace.WithSpanKind(trace.SpanKindServer),
		},
	}

	requestCountCounter, err := meter.Int64Counter(
		requestCountCounterName,
		metric.WithDescription("Measures the number of HTTP requests."),
	)
	must(err)
	o.requestCountCounter = requestCountCounter

	requestSizeHistogram, err := meter.Int64Histogram(
		requestSizeHistogramName,
		metric.WithUnit("By"),
		metric.WithDescription("Measures the size of HTTP request bodies."),
		metric.WithExplicitBucketBoundaries(0, 100, 1000, 10000, 100000, 1000000, 10000000, 100000000),
	)
	must(err)
	o.requestSizeHistogram = requestSizeHistogram

	requestDurationHistogram, err := meter.Float64Histogram(
		requestDurationHistogramName,
		metric.WithUnit("ms"),
		metric.WithDescription("Measures the time spent handling HTTP requests."),
		metric.WithExplicitBucketBoundaries(1, 10, 100, 1000, 10000),
	)
	must(err)
	o.requestDurationHistogram = requestDurationHistogram

	responseCounter, err := meter.Int64Counter(
		responseCounterName,
		metric.WithDescription("Measures HTTP responses."),
	)
	must(err)
	o.responseCounter = responseCounter

	responseSizeHistogram, err := meter.Int64Histogram(
		responseSizeHistogramName,
		metric.WithUnit("By"),
		metric.WithDescription("Measures the size of HTTP response bodies."),
		metric.WithExplicitBucketBoundaries(0, 100, 1000, 10000, 100000, 1000000, 10000000, 100000000),
	)
	must(err)
	o.responseSizeHistogram = responseSizeHistogram

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Check request trace context
		propagator := otel.GetTextMapPropagator()
		ctx := propagator.Extract(r.Context(), propagation.HeaderCarrier(r.Header))

		var tracer trace.Tracer
		if span := trace.SpanFromContext(r.Context()); span.SpanContext().IsValid() {
			tracer = o.newTracer(span.TracerProvider())
		} else {
			tracer = o.newTracer(otel.GetTracerProvider())
		}

		// Start a span here to include ALL operations of KRB
		ctx, span := tracer.Start(ctx, spanName, o.spanOpts...)
		defer span.End() // Stop the span after EVERYTHING is done

		logger.Info(r.Method + " " + r.URL.Path)

		// Wrap the request body to extract size
		// Wrap the response to extract:
		// - status code
		// - response body size
		bw := newBodyWrapper(r.Body).(*bodyWrapper)
		if r.Body != nil && r.Body != http.NoBody {
			r.Body = bw
		}

		w, rw := newResponseWrapper(w)

		// Since the duration metric is directly related to the route forwarded to, keep the time
		// measurement as close to the forwarding call as possible.
		start := time.Now()
		next.ServeHTTP(w, r.WithContext(ctx))
		duration := time.Since(start)

		// Process the response.
		span.SetStatus(rw.SpanStatus())

		// Update all metrics, can't separate request and response handling since the handler is
		// called by ServeHTTP.
		// TODO: add backend selection
		// TODO: add route
		statusCodeOpt := metric.WithAttributeSet(attribute.NewSet(semconv.HTTPStatusCode(int(rw.StatusCode()))))

		// Request
		o.requestCountCounter.Add(ctx, 1)
		o.requestSizeHistogram.Record(ctx, int64(bw.NumBytes()))
		o.requestDurationHistogram.Record(ctx, float64(duration/time.Millisecond))

		// Response
		o.responseCounter.Add(ctx, 1, statusCodeOpt)
		o.responseSizeHistogram.Record(ctx, int64(rw.NumBytes()))

		logger.Info(r.Method + " " + r.URL.Path)
	})
}

func (o *obs) newTracer(provider trace.TracerProvider) trace.Tracer {
	return provider.Tracer(tracerName)
}

func must(err error) {
	if err != nil {
		panic(err)
	}
}
