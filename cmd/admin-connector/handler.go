package main

import (
	"net/http"
	"net/http/httputil"
	"net/url"

	apierror "github.com/trebent/kerberos/internal/oapi/error"
	"github.com/trebent/kerberos/internal/security"
)

type (
	opts struct {
		target string
		scheme string
	}
	connectorHandler struct {
		proxy *httputil.ReverseProxy
	}
)

func newHandler(opts opts) *connectorHandler {
	u := &url.URL{
		Scheme: opts.scheme,
		Host:   opts.target,
	}
	return &connectorHandler{
		proxy: httputil.NewSingleHostReverseProxy(u),
	}
}

func (h *connectorHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	sessionCookies := r.CookiesNamed(security.SessionCookieName)

	if len(sessionCookies) == 0 {
		// No session cookie found, return a 401 Unauthorized response.
		apierror.ErrorHandler(w, r, apierror.ErrUnauthorized)
		return
	}

	h.proxy.ServeHTTP(w, r)
}
