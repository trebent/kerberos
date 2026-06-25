// nolint: mnd
package obs

import (
	"context"
	"net/http"
	"strconv"
	"time"

	"github.com/go-logr/logr"
	"github.com/trebent/kerberos/internal/composer"
	"github.com/trebent/kerberos/internal/composer/debug"
	"github.com/trebent/kerberos/internal/composer/router"
	"github.com/trebent/kerberos/internal/config"
	adminapi "github.com/trebent/kerberos/internal/oapi/admin"
	apierror "github.com/trebent/kerberos/internal/oapi/error"
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
		next composer.FlowComponent
		cfg  *config.ObservabilityConfig

		logger   logr.Logger
		debugger debug.Debugger

		spanOpts                 []trace.SpanStartOption
		requestCounter           metric.Int64Counter
		requestSizeHistogram     metric.Int64Histogram
		requestDurationHistogram metric.Float64Histogram
		responseCounter          metric.Int64Counter
		responseSizeHistogram    metric.Int64Histogram
	}
	Opts struct {
		Cfg *config.ObservabilityConfig

		Version string

		Debugger debug.Debugger
	}
)

const (
	tracerName = "krb"

	requestCounterName           = "request.count"
	requestSizeHistogramName     = "request.size"
	requestDurationHistogramName = "request.duration"

	responseCounterName       = "response"
	responseSizeHistogramName = "response.size"
)

// nolint: gochecknoglobals
var (
	tracer                        = otel.Tracer(tracerName)
	_      composer.FlowComponent = (*obs)(nil)
)

func NewComponent(opts *Opts) composer.FlowComponent {
	logger := zerologr.WithName("request")

	if !opts.Cfg.Enabled {
		return dummyComponent(logger, opts)
	}

	meter := otel.GetMeterProvider().Meter(
		"github.com/trebent/kerberos",
		metric.WithInstrumentationVersion(opts.Version),
	)

	requestCountCounter, err := meter.Int64Counter(
		requestCounterName,
		metric.WithDescription("Measures the number of HTTP requests."),
	)
	must(err)

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

	requestDurationHistogram, err := meter.Float64Histogram(
		requestDurationHistogramName,
		metric.WithUnit("ms"),
		metric.WithDescription("Measures the time spent handling HTTP requests."),
		metric.WithExplicitBucketBoundaries(1, 10, 100, 1000, 10000),
	)
	must(err)

	responseCounter, err := meter.Int64Counter(
		responseCounterName,
		metric.WithDescription("Keeps track of HTTP response status code counts."),
	)
	must(err)

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

	return &obs{
		spanOpts: []trace.SpanStartOption{
			trace.WithSpanKind(trace.SpanKindServer),
		},
		cfg:      opts.Cfg,
		logger:   logger,
		debugger: opts.Debugger,

		requestCounter:           requestCountCounter,
		requestSizeHistogram:     requestSizeHistogram,
		requestDurationHistogram: requestDurationHistogram,
		responseCounter:          responseCounter,
		responseSizeHistogram:    responseSizeHistogram,
	}
}

// Next implements [composer.FlowComponent].
func (o *obs) Next(next composer.FlowComponent) {
	o.next = next
}

// GetMeta implements [composer.FlowComponent].
func (o *obs) GetMeta() []adminapi.FlowMeta {
	fmd := adminapi.FlowMeta_Data{}
	if err := fmd.FromFlowMetaDataObservability(
		adminapi.FlowMetaDataObservability{Enabled: o.cfg.Enabled},
	); err != nil {
		panic(err)
	}

	return append([]adminapi.FlowMeta{
		{
			Name: "observability",
			Data: fmd,
		},
	}, o.next.GetMeta()...)
}

func (o *obs) spanStartOpts(req *http.Request) []trace.SpanStartOption {
	opts := make([]trace.SpanStartOption, len(o.spanOpts)+2)
	copy(opts, o.spanOpts)
	opts[len(opts)-1] = trace.WithAttributes(semconv.HTTPMethod(req.Method))
	opts[len(opts)-2] = trace.WithAttributes(semconv.HTTPURL(req.URL.Path))

	return opts
}

// ServeHTTP implements [types.FlowComponent].
func (o *obs) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	debugStart := time.Now()

	// Check request trace context
	ctx := otel.GetTextMapPropagator().Extract(req.Context(), propagation.HeaderCarrier(req.Header))
	ctx, span := tracer.Start(ctx, req.Method, o.spanStartOpts(req)...)
	defer span.End() // Stop the span after EVERYTHING is done

	rLogger := o.logger.WithValues("path", req.URL.Path, "method", req.Method)
	originalPath := req.URL.Path
	rLogger.Info(req.Method + " " + originalPath)

	// Wrap the response to extract:
	// - status code
	// - response body size
	wrapped := response.NewResponseWrapper(w)

	// Wrapped body to extract size.
	bw, _ := response.NewBodyWrapper(req.Body).(*response.BodyWrapper)

	// Extract the backend backendName to enable debugging and context enrichment early on.
	backendName, err := router.GetBackendName(req)
	if err != nil {
		rLogger.Error(err, "Failed to extract backend name from request path")
		apierror.ErrorHandler(wrapped, req, err)
		krbAttributes := extractKrbAttributes(ctx)
		//nolint:errcheck // no point
		o.bumpMetrics(ctx, wrapped.(*response.Wrapper), bw, req, 0, krbAttributes)

		span.SetStatus(http.StatusBadRequest, http.StatusText(http.StatusBadRequest))
		span.SetAttributes(krbAttributes...)

		// Debugging this failure is pointless since the session matching will inevitably
		return
	}

	// Make sure this is set prior to debugging, always.
	ctx = context.WithValue(ctx, composer.BackendContextKey, backendName)

	// Debug call is started.
	debugCall, ctx := o.debugger.Start(ctx)
	defer debugCall.Finalise()
	debugCall.SetURL(req.URL.Path)
	debugCall.SetMethod(req.Method)

	ctx = logr.NewContext(ctx, rLogger)

	// Wrap the request body to extract size
	if req.Body != nil && req.Body != http.NoBody {
		req.Body = bw
	}

	debugCall.AddTransition(
		"obs",
		debug.CallDirectionInbound,
		debugStart,
		time.Now(),
		debug.CallResultSuccess,
		"",
	)

	// Since the duration metric is directly related to the route forwarded to, keep the time
	// measurement as close to the forwarding call as possible.
	start := time.Now()
	o.next.ServeHTTP(wrapped, req.WithContext(ctx))

	// Keep this as close to the forwarding call as possible to measure
	// the duration of the request handling.
	duration := time.Since(start)

	// Reset component start to measure response handling.
	debugStart = time.Now()

	// Process the response, update the span and metrics with attributes.
	wrapper, _ := wrapped.(*response.Wrapper)
	krbAttributes := extractKrbAttributes(wrapper.GetRequestContext())

	o.bumpMetrics(ctx, wrapper, bw, req, duration, krbAttributes)

	span.SetStatus(wrapper.SpanStatus())
	span.SetAttributes(krbAttributes...)

	rLogger.Info(
		req.Method+" "+originalPath+" "+strconv.Itoa(wrapper.StatusCode()),
		string(semconv.HTTPStatusCodeKey), wrapper.StatusCode(),
	)

	debugCall.SetStatusCode(wrapper.StatusCode())
	debugCall.AddTransition(
		"obs",
		debug.CallDirectionOutbound,
		debugStart,
		time.Now(),
		debug.CallResultSuccess,
		"",
	)
}

func (o *obs) bumpMetrics(
	ctx context.Context,
	wrapper *response.Wrapper,
	bw *response.BodyWrapper,
	req *http.Request,
	duration time.Duration,
	attributes []attribute.KeyValue,
) {
	// Update metrics, can't separate request and response handling since the handler is
	// called by ServeHTTP, no
	statusCodeOpt := metric.WithAttributes(semconv.HTTPStatusCode(wrapper.StatusCode()))
	requestMeta := metric.WithAttributes(semconv.HTTPMethod(req.Method))
	krbMetricMeta := metric.WithAttributes(attributes...)

	// Request
	o.requestCounter.Add(ctx, 1, requestMeta, krbMetricMeta)
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
}

func must(err error) {
	if err != nil {
		panic(err)
	}
}

func extractKrbAttributes(ctx context.Context) []attribute.KeyValue {
	attributes := make([]attribute.KeyValue, 0, 1)
	backend := ctx.Value(composer.BackendContextKey)
	if backend == nil {
		attributes = append(attributes, attribute.String("krb.backend", "unknown"))
	} else {
		//nolint:errcheck
		attributes = append(attributes, attribute.String("krb.backend", backend.(string)))
	}

	return attributes
}

func dummyComponent(logger logr.Logger, opts *Opts) composer.FlowComponent {
	logger.Info("Observability has been disabled, setting dummy component for logging")

	// Observability disabled still logs and debugs incoming requests, ensuring that the request
	// context contains the expected values for downstream components.
	return &composer.Dummy{CustomHandler: func(
		next composer.FlowComponent,
		w http.ResponseWriter,
		req *http.Request,
	) {
		rLogger := logger.WithValues("path", req.URL.Path, "method", req.Method)
		rLogger.Info(req.Method + " " + req.URL.Path)

		// Debug call is started, but the flow component transition is not logged to denote that
		// observability is indeed disabled.
		debugCall, ctx := opts.Debugger.Start(req.Context())
		defer debugCall.Finalise()
		debugCall.SetURL(req.URL.Path)
		debugCall.SetMethod(req.Method)

		name, err := router.GetBackendName(req)
		if err != nil {
			rLogger.Error(err, "Failed to extract backend name from request path")
			apierror.ErrorHandler(w, req, err)
			return
		}

		ctx = context.WithValue(ctx, composer.BackendContextKey, name)
		ctx = logr.NewContext(ctx, rLogger)

		// Must set up response wrapper since components down the line depends on it, and to
		// capture status code.
		wrapped := response.NewResponseWrapper(w)
		//nolint:errcheck // no point
		wrapper := wrapped.(*response.Wrapper)

		// Handle the call by forwarding to the next component in the flow.
		next.ServeHTTP(wrapper, req.WithContext(ctx))

		// Set debugging metadata for the response, including status code and log the request.
		debugCall.SetStatusCode(wrapper.StatusCode())
		rLogger.Info(
			req.Method+" "+req.URL.Path+" "+strconv.Itoa(wrapper.StatusCode()),
			string(semconv.HTTPStatusCodeKey), wrapper.StatusCode(),
		)
	}}
}
