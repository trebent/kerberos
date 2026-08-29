// nolint: mnd
package obs

import (
	"context"
	"fmt"
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
	intotel "github.com/trebent/kerberos/internal/otel"
	"github.com/trebent/kerberos/internal/response"
	"github.com/trebent/zerologr"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
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

		tracer   trace.Tracer
		spanOpts []trace.SpanStartOption

		metrics *intotel.StdHTTPMetrics
	}
	Opts struct {
		Cfg *config.ObservabilityConfig

		Version string

		Debugger debug.Debugger
	}
)

const (
	tracerName = "krb"
)

// nolint: gochecknoglobals
var _ composer.FlowComponent = (*obs)(nil)

func NewComponent(opts *Opts) composer.FlowComponent {
	logger := zerologr.WithName("request")

	if !opts.Cfg.Enabled {
		return dummyComponent(logger, opts)
	}

	metrics, err := intotel.StandardHTTPMetrics("", opts.Version)
	must(err)

	return &obs{
		tracer: otel.Tracer(tracerName, trace.WithInstrumentationVersion(opts.Version)),
		spanOpts: []trace.SpanStartOption{
			trace.WithSpanKind(trace.SpanKindServer),
		},
		cfg:      opts.Cfg,
		logger:   logger,
		debugger: opts.Debugger,

		metrics: metrics,
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
			Name: "obs",
			Data: fmd,
		},
	}, o.next.GetMeta()...)
}

func (o *obs) spanStartOpts(req *http.Request) []trace.SpanStartOption {
	opts := make([]trace.SpanStartOption, len(o.spanOpts)+2)
	copy(opts, o.spanOpts)
	opts[len(opts)-1] = trace.WithAttributes(semconv.HTTPMethod(req.Method))
	opts[len(opts)-2] = trace.WithAttributes(semconv.HTTPURL(req.URL.String()))

	return opts
}

// ServeHTTP implements [types.FlowComponent].
func (o *obs) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	debugStart := time.Now()

	// Check request trace context
	ctx := otel.GetTextMapPropagator().Extract(req.Context(), propagation.HeaderCarrier(req.Header))
	ctx, span := o.tracer.Start(
		ctx,
		fmt.Sprintf("%s %s", req.Method, req.URL.String()),
		o.spanStartOpts(req)...,
	)
	defer span.End() // Stop the span after EVERYTHING is done

	rLogger := o.logger.WithValues("http.path", req.URL.Path, semconv.HTTPMethodKey, req.Method)
	originalPath := req.URL.Path

	// Wrap the response to extract:
	// - status code
	// - response body size
	wrapped := response.NewResponseWrapper(w)

	// Extract the backend backendName to enable debugging and context enrichment early on.
	backendName, err := router.GetBackendName(req)
	if err != nil {
		rLogger.Error(err, "Failed to extract backend name from request path")
		apierror.ErrorHandler(wrapped, req, err)
		krbAttributes := extractKrbAttributes(ctx)
		//nolint:errcheck // no point
		o.metrics.Bump(ctx, wrapped.(*response.Wrapper), nil, req, 0, krbAttributes...)

		span.SetStatus(codes.Error, http.StatusText(http.StatusBadRequest))
		span.SetAttributes(krbAttributes...)

		rLogger.Info(
			req.Method + " " + originalPath + " " + strconv.Itoa(http.StatusBadRequest),
		)

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

	var bw *response.BodyWrapper
	// Wrap the request body to extract size
	if req.Body != nil && req.Body != http.NoBody {
		// Wrapped body to extract size.
		bw, _ = response.NewBodyWrapper(req.Body).(*response.BodyWrapper)
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

	o.metrics.Bump(ctx, wrapper, bw, req, duration, krbAttributes...)

	span.SetStatus(wrapper.SpanStatus())
	span.SetAttributes(krbAttributes...)

	rLogger.Info(
		req.Method + " " + originalPath + " " + strconv.Itoa(wrapper.StatusCode()),
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

		name, err := router.GetBackendName(req)
		if err != nil {
			rLogger.Error(err, "Failed to extract backend name from request path")
			apierror.ErrorHandler(w, req, err)
			return
		}

		ctx := context.WithValue(req.Context(), composer.BackendContextKey, name)

		// Debug call is started, but the flow component transition is not logged to denote that
		// observability is indeed disabled.
		debugCall, ctx := opts.Debugger.Start(ctx)
		defer debugCall.Finalise()
		debugCall.SetURL(req.URL.Path)
		debugCall.SetMethod(req.Method)

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
