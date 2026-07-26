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
	sessionCookies := r.CookiesNamed(security.SessionCookieName)

	// #1 -> are there any session cookies? If not, return 401 Unauthorized.
	if len(sessionCookies) == 0 {
		// No session cookie found, return a 401 Unauthorized response.
		apierror.ErrorHandler(w, r, apierror.ErrUnauthorized)
		return
	}

	// #2 -> is the session cookie referring to an existing session? If not, return 401 Unauthorized.
	session, err := admindb.GetSession(r.Context(), h.sqlClient, sessionCookies[0].Value)
	if errors.Is(err, db.ErrRowNotFound) {
		apierror.ErrorHandler(w, r, apierror.ErrUnauthorized)
		return
	}

	// #3 -> is the session cookie expired? If so, return 401 Unauthorized.
	if time.Until(time.UnixMilli(session.Expires)) <= 0 {
		apierror.ErrorHandler(w, r, apierror.ErrUnauthorized)
		return
	}

	h.proxy.ServeHTTP(w, r)
}
