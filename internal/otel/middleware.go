// nolint: mnd
package otel

import (
	"net/http"
	"strconv"
	"time"

	"github.com/go-logr/logr"
	"github.com/trebent/kerberos/internal/response"
	"github.com/trebent/kerberos/internal/router"
	"github.com/trebent/kerberos/internal/version"
	"github.com/trebent/zerologr"
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

func Middleware(next http.Handler) http.Handler {
	o := newObs()
	logger := zerologr.WithName("request")

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

		rLogger := logger.WithValues("path", r.URL.Path, "method", r.Method)
		rLogger.Info(r.Method + " " + r.URL.Path)
		ctx = logr.NewContext(ctx, rLogger)

		// Wrap the request body to extract size
		bw, _ := response.NewBodyWrapper(r.Body).(*response.BodyWrapper)
		if r.Body != nil && r.Body != http.NoBody {
			r.Body = bw
		}

		// Wrap the response to extract:
		// - status code
		// - response body size
		wrapped := response.NewResponseWrapper(w)
		wrapper, _ := wrapped.(*response.ResponseWrapper)

		// Update at least the request counter here, since it's not reliant on handler logic.
		o.requestCountCounter.Add(ctx, 1)

		// Since the duration metric is directly related to the route forwarded to, keep the time
		// measurement as close to the forwarding call as possible.
		start := time.Now()
		next.ServeHTTP(wrapped, r.WithContext(ctx))
		duration := time.Since(start)

		// Process the response.
		span.SetStatus(wrapper.SpanStatus())

		// Update metrics, can't separate request and response handling since the handler is
		// called by ServeHTTP, no
		statusCodeOpt := metric.WithAttributes(semconv.HTTPStatusCode(wrapper.StatusCode()))
		generalOpts := metric.WithAttributes(
			semconv.HTTPMethod(r.Method),
			semconv.HTTPRoute(r.URL.Path),
			attribute.String("krb.backend", router.BackendFromContext(wrapper.GetRequestContext()).Name()),
		)
		// TODO: add actual route

		// Request
		o.requestSizeHistogram.Record(ctx, bw.NumBytes(), generalOpts)
		o.requestDurationHistogram.Record(ctx, float64(duration/time.Millisecond), generalOpts)

		// Response
		o.responseCounter.Add(ctx, 1, statusCodeOpt, generalOpts)
		o.responseSizeHistogram.Record(ctx, wrapper.NumBytes(), generalOpts)

		rLogger.Info(r.Method + " " + r.URL.Path + " " + strconv.Itoa(wrapper.StatusCode()))
	})
}

func newObs() *obs {
	o := &obs{
		spanOpts: []trace.SpanStartOption{
			trace.WithSpanKind(trace.SpanKindServer),
		},
	}

	meter := otel.GetMeterProvider().Meter(
		"github.com/trebent/kerberos/forwarder",
		metric.WithInstrumentationVersion(version.Version()),
	)

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
		metric.WithDescription("Keeps track of HTTP response status code counts."),
	)
	must(err)
	o.responseCounter = responseCounter

	responseSizeHistogram, err := meter.Int64Histogram(
		responseSizeHistogramName,
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
	must(err)
	o.responseSizeHistogram = responseSizeHistogram

	return o
}

func (o *obs) newTracer(provider trace.TracerProvider) trace.Tracer {
	return provider.Tracer(tracerName)
}

func must(err error) {
	if err != nil {
		panic(err)
	}
}
