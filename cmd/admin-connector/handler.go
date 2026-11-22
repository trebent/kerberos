package main

import (
	"crypto/tls"
	"crypto/x509"
	"errors"
	"fmt"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"time"

	admindb "github.com/trebent/kerberos/internal/admin/db"
	"github.com/trebent/kerberos/internal/config"
	"github.com/trebent/kerberos/internal/db"
	apierror "github.com/trebent/kerberos/internal/oapi/error"
	"github.com/trebent/kerberos/internal/security"
	"github.com/trebent/zerologr"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/propagation"
	semconv "go.opentelemetry.io/otel/semconv/v1.20.0"
	"go.opentelemetry.io/otel/trace"
)

type (
	opts struct {
		version   string
		target    string
		scheme    string
		sqlClient db.SQLClient
		targetTLS *config.TargetTLS
	}
	connectorHandler struct {
		proxy     *httputil.ReverseProxy
		sqlClient db.SQLClient

		callCounter           metric.Int64Counter
		callDeniedCounter     metric.Int64Counter
		calloutFailureCounter metric.Int64Counter
	}
)

var (
	tracer = otel.GetTracerProvider().Tracer("admin-connector")

	// errSessionDenied marks session check failures caused by the request itself,
	// as opposed to failures to look the session up.
	errSessionDenied = errors.New("session denied")
)

func newHandler(opts opts) (*connectorHandler, error) {
	u := &url.URL{
		Scheme: opts.scheme,
		Host:   opts.target,
	}
	transport, err := makeTransport(opts)
	if err != nil {
		return nil, fmt.Errorf("building proxy transport: %w", err)
	}
	ch := &connectorHandler{
		proxy: &httputil.ReverseProxy{
			Transport: transport,
			Rewrite: func(r *httputil.ProxyRequest) {
				r.SetURL(u)
				security.StripKerberosCookies(r.Out)
			},
		},
		sqlClient: opts.sqlClient,
	}
	ch.proxy.ErrorHandler = func(w http.ResponseWriter, r *http.Request, err error) {
		zerologr.Error(err, "Error proxying request")
		ch.calloutFailureCounter.Add(r.Context(), 1)
		apierror.ErrorHandler(w, r, apierror.ErrBadGateway)
	}

	return ch, ch.initMetrics(opts)
}

func (h *connectorHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	ctx := otel.GetTextMapPropagator().Extract(r.Context(), propagation.HeaderCarrier(r.Header))

	ctx, newSpan := tracer.Start(
		ctx,
		fmt.Sprintf("%s %s", r.Method, r.URL.String()),
		trace.WithSpanKind(trace.SpanKindServer),
		trace.WithAttributes(
			semconv.HTTPMethod(r.Method),
			semconv.HTTPURL(r.URL.String()),
		),
	)
	defer newSpan.End()

	if err := h.checkSession(r); err != nil {
		zerologr.Error(err, "Session check failed")
		apiErr := apierror.ErrISE
		if errors.Is(err, errSessionDenied) {
			apiErr = apierror.ErrUnauthorized
			h.callDeniedCounter.Add(ctx, 1)
		}
		newSpan.SetStatus(codes.Error, apiErr.Error())
		apierror.ErrorHandler(w, r, apiErr)
		return
	}

	otel.GetTextMapPropagator().
		Inject(ctx, propagation.HeaderCarrier(r.Header))

	h.callCounter.Add(ctx, 1)
	newSpan.SetStatus(codes.Ok, "proxied")
	h.proxy.ServeHTTP(w, r.WithContext(ctx))
}

func makeTransport(opts opts) (http.RoundTripper, error) {
	if opts.targetTLS == nil {
		return http.DefaultTransport, nil
	}

	tc := &tls.Config{
		//nolint:gosec // InsecureSkipVerify is an explicit opt-in for non-production environments.
		InsecureSkipVerify: opts.targetTLS.InsecureSkipVerify,
	}

	if opts.targetTLS.RootCAFile != "" {
		zerologr.Info("Loading target root CA bundle", "path", opts.targetTLS.RootCAFile)
		pem, err := os.ReadFile(opts.targetTLS.RootCAFile)
		if err != nil {
			return nil, fmt.Errorf("reading root CA bundle %q: %w", opts.targetTLS.RootCAFile, err)
		}
		pool := x509.NewCertPool()
		if !pool.AppendCertsFromPEM(pem) {
			return nil, fmt.Errorf(
				"no valid PEM certificates found in %q",
				opts.targetTLS.RootCAFile,
			)
		}
		tc.RootCAs = pool
	}

	return &http.Transport{TLSClientConfig: tc}, nil
}

func (h *connectorHandler) checkSession(r *http.Request) error {
	sessionCookies := r.CookiesNamed(security.SessionCookieName)
	// #1 -> are there any session cookies? If not, return 401 Unauthorized.
	if len(sessionCookies) == 0 {
		return fmt.Errorf("%w: no session cookie found in request", errSessionDenied)
	}

	// #2 -> is the session cookie referring to an existing session? If not, return 401 Unauthorized.
	session, err := admindb.GetSession(r.Context(), h.sqlClient, sessionCookies[0].Value)
	if errors.Is(err, db.ErrRowNotFound) {
		return fmt.Errorf("%w: %w", errSessionDenied, err)
	}
	if err != nil {
		return fmt.Errorf("looking up session: %w", err)
	}

	// #3 -> is the session cookie expired? If so, return 401 Unauthorized.
	if time.Until(time.UnixMilli(session.Expires)) <= 0 {
		return fmt.Errorf("%w: session cookie is expired", errSessionDenied)
	}

	return nil
}

func (h *connectorHandler) initMetrics(opts opts) error {
	meter := otel.GetMeterProvider().Meter(
		"github.com/trebent/kerberos/admin-connector",
		metric.WithInstrumentationVersion(opts.version),
	)

	var err error

	h.callCounter, err = meter.Int64Counter(
		"admin_connector_calls_total",
		metric.WithDescription("Total number of calls to the admin connector"),
	)
	if err != nil {
		return fmt.Errorf("failed to create call counter: %w", err)
	}

	h.callDeniedCounter, err = meter.Int64Counter(
		"admin_connector_calls_denied_total",
		metric.WithDescription("Total number of denied calls to the admin connector"),
	)
	if err != nil {
		return fmt.Errorf("failed to create call denied counter: %w", err)
	}

	h.calloutFailureCounter, err = meter.Int64Counter(
		"admin_connector_callout_failures_total",
		metric.WithDescription("Total number of callout failures from the admin connector"),
	)
	if err != nil {
		return fmt.Errorf("failed to create callout failure counter: %w", err)
	}

	return nil
}
