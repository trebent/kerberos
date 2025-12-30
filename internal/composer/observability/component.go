// nolint: mnd
package obs

import (
	"context"
	"net/http"
	"strconv"
	"time"

	"github.com/go-logr/logr"
	composertypes "github.com/trebent/kerberos/internal/composer/types"
	"github.com/trebent/kerberos/internal/config"
	"github.com/trebent/kerberos/internal/env"
	"github.com/trebent/kerberos/internal/response"
	"github.com/trebent/zerologr"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/propagation"
	semconv "go.opentelemetry.io/otel/semconv/v1.20.0"
	"go.opentelemetry.io/otel/trace"
)

type (
	obs struct {
		next composertypes.FlowComponent
		cfg  *obsConfig

		logger                   logr.Logger
		spanOpts                 []trace.SpanStartOption
		requestCountCounter      metric.Int64Counter
		requestSizeHistogram     metric.Int64Histogram
		requestDurationHistogram metric.Float64Histogram
		responseCounter          metric.Int64Counter
		responseSizeHistogram    metric.Int64Histogram
	}
	Opts struct {
		Cfg config.Map
	}
)

const (
	tracerName = "krb"

	requestCountCounterName      = "request.count"
	requestSizeHistogramName     = "request.size"
	requestDurationHistogramName = "request.duration"

	responseCounterName       = "response"
	responseSizeHistogramName = "response.size"
)

// nolint: gochecknoglobals
var (
	tracer                             = otel.Tracer(tracerName)
	_      composertypes.FlowComponent = (*obs)(nil)
)

func NewComponent(opts *Opts) composertypes.FlowComponent {
	o := &obs{
		spanOpts: []trace.SpanStartOption{
			trace.WithSpanKind(trace.SpanKindServer),
		},
		cfg: config.AccessAs[*obsConfig](opts.Cfg, configName),
	}

	o.logger = zerologr.WithName("request")

	meter := otel.GetMeterProvider().Meter(
		"github.com/trebent/kerberos",
		metric.WithInstrumentationVersion(env.Version.Value()),
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

// Next implements [types.FlowComponent].
func (o *obs) Next(next composertypes.FlowComponent) {
	o.next = next
}

// ServeHTTP implements [types.FlowComponent].
func (o *obs) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	// Check request trace context
	ctx := otel.GetTextMapPropagator().Extract(req.Context(), propagation.HeaderCarrier(req.Header))

	// Start a span here to include ALL operations of KRB
	// TODO: add more to the span name?
	ctx, span := tracer.Start(ctx, req.Method, o.spanOpts...)
	defer span.End() // Stop the span after EVERYTHING is done

	rLogger := o.logger.WithValues("path", req.URL.Path, "method", req.Method)
	rLogger.Info(req.Method + " " + req.URL.Path)
	ctx = logr.NewContext(ctx, rLogger)

	// Wrap the request body to extract size
	bw, _ := response.NewBodyWrapper(req.Body).(*response.BodyWrapper)
	if req.Body != nil && req.Body != http.NoBody {
		req.Body = bw
	}

	// Wrap the response to extract:
	// - status code
	// - response body size
	wrapped := response.NewResponseWrapper(w)

	// Since the duration metric is directly related to the route forwarded to, keep the time
	// measurement as close to the forwarding call as possible.
	start := time.Now()
	o.next.ServeHTTP(wrapped, req.WithContext(ctx))
	duration := time.Since(start)

	// Process the response, update the span with attributes.
	wrapper, _ := wrapped.(*response.Wrapper)
	krbAttributes := extractKrbAttributes(wrapper.GetRequestContext())

	span.SetStatus(wrapper.SpanStatus())
	span.SetAttributes(krbAttributes...)

	// Update metrics, can't separate request and response handling since the handler is
	// called by ServeHTTP, no
	statusCodeOpt := metric.WithAttributes(semconv.HTTPStatusCode(wrapper.StatusCode()))
	requestMeta := metric.WithAttributes(semconv.HTTPMethod(req.Method))
	krbMetricMeta := metric.WithAttributes(krbAttributes...)

	// Request
	o.requestCountCounter.Add(ctx, 1, requestMeta, krbMetricMeta)
	o.requestSizeHistogram.Record(ctx, bw.NumBytes(), requestMeta, krbMetricMeta)
	o.requestDurationHistogram.Record(
		ctx,
		float64(duration/time.Millisecond),
		requestMeta,
		krbMetricMeta,
	)

	// Response
	o.responseCounter.Add(ctx, 1, statusCodeOpt, requestMeta, krbMetricMeta)
	o.responseSizeHistogram.Record(ctx, wrapper.NumBytes(), requestMeta, krbMetricMeta)

	rLogger.Info(
		req.Method+" "+req.URL.Path+" "+strconv.Itoa(wrapper.StatusCode()),
		string(semconv.HTTPStatusCodeKey), wrapper.StatusCode(),
	)
}

func must(err error) {
	if err != nil {
		panic(err)
	}
}

func extractKrbAttributes(ctx context.Context) []attribute.KeyValue {
	attributes := make([]attribute.KeyValue, 0, 1)
	backend := ctx.Value(composertypes.BackendContextKey)
	if backend == nil {
		attributes = append(attributes, attribute.String("krb.backend", "unknown"))
	} else {
		//nolint:errcheck
		attributes = append(attributes, attribute.String("krb.backend", backend.(string)))
	}

	return attributes
}
