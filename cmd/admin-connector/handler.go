package main

import (
	"errors"
	"net/http"
	"net/http/httputil"
	"net/url"
	"time"

	admindb "github.com/trebent/kerberos/internal/admin/db"
	"github.com/trebent/kerberos/internal/db"
	apierror "github.com/trebent/kerberos/internal/oapi/error"
	"github.com/trebent/kerberos/internal/security"
	"github.com/trebent/zerologr"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/propagation"
	semconv "go.opentelemetry.io/otel/semconv/v1.20.0"
	"go.opentelemetry.io/otel/trace"
)

type (
	opts struct {
		target    string
		scheme    string
		sqlClient db.SQLClient
	}
	connectorHandler struct {
		proxy     *httputil.ReverseProxy
		sqlClient db.SQLClient
	}
)

var tracer = otel.GetTracerProvider().Tracer("admin-connector")

func newHandler(opts opts) *connectorHandler {
	u := &url.URL{
		Scheme: opts.scheme,
		Host:   opts.target,
	}
	return &connectorHandler{
		proxy:     httputil.NewSingleHostReverseProxy(u),
		sqlClient: opts.sqlClient,
	}
}

func (h *connectorHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	ctx := otel.GetTextMapPropagator().Extract(r.Context(), propagation.HeaderCarrier(r.Header))

	ctx, newSpan := tracer.Start(
		ctx,
		"proxying",
		trace.WithSpanKind(trace.SpanKindServer),
		trace.WithAttributes(
			semconv.HTTPMethod(r.Method),
			semconv.HTTPURL(r.URL.String()),
		),
	)
	defer newSpan.End()

	sessionCookies := r.CookiesNamed(security.SessionCookieName)

	// #1 -> are there any session cookies? If not, return 401 Unauthorized.
	if len(sessionCookies) == 0 {
		zerologr.V(20).Info("No session cookie found in request")
		newSpan.SetStatus(codes.Error, http.StatusText(http.StatusUnauthorized))
		// No session cookie found, return a 401 Unauthorized response.
		apierror.ErrorHandler(w, r, apierror.ErrUnauthorized)
		return
	}

	// #2 -> is the session cookie referring to an existing session? If not, return 401 Unauthorized.
	session, err := admindb.GetSession(r.Context(), h.sqlClient, sessionCookies[0].Value)
	if errors.Is(err, db.ErrRowNotFound) {
		zerologr.V(20).Info("No session found")
		newSpan.SetStatus(codes.Error, http.StatusText(http.StatusUnauthorized))
		apierror.ErrorHandler(w, r, apierror.ErrUnauthorized)
		return
	}

	// #3 -> is the session cookie expired? If so, return 401 Unauthorized.
	if time.Until(time.UnixMilli(session.Expires)) <= 0 {
		zerologr.V(20).Info("Session expired")
		newSpan.SetStatus(codes.Error, http.StatusText(http.StatusUnauthorized))
		apierror.ErrorHandler(w, r, apierror.ErrUnauthorized)
		return
	}

	h.proxy.ServeHTTP(w, r.WithContext(ctx))
}
