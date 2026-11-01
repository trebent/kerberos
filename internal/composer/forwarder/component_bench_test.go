package forwarder

import (
	"context"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/go-logr/logr"
	"github.com/trebent/kerberos/internal/composer"
	"github.com/trebent/kerberos/internal/config"
	"github.com/trebent/zerologr"
)

const benchmarkPayload = "somebytessomebytessomebytessomebytessomebytessomebytessomebytessomebytes" +
	"somebytessomebytessomebytessomebytessomebytessomebytessomebytessomebytes"

func BenchmarkForwarder(b *testing.B) {
	zerologr.Set(logr.Discard())

	b.Run("no body", func(b *testing.B) {
		comp, ctx := newBenchmarkForwarder(b)
		recorder := httptest.NewRecorder()
		request := httptest.NewRequestWithContext(ctx, http.MethodGet, "/", http.NoBody)

		b.ReportAllocs()
		for b.Loop() {
			// Start each iteration with an empty header map, like a fresh response writer.
			clear(recorder.Header())
			comp.ServeHTTP(recorder, request)
		}

		checkStatus(b, recorder)
	})

	b.Run("body", func(b *testing.B) {
		comp, ctx := newBenchmarkForwarder(b)
		recorder := httptest.NewRecorder()
		request := httptest.NewRequestWithContext(ctx, http.MethodGet, "/", http.NoBody)
		// http.Server sets this from the Content-Length header of incoming requests.
		request.ContentLength = int64(len(benchmarkPayload))

		b.ReportAllocs()
		for b.Loop() {
			clear(recorder.Header())
			// The transport may still be reading or closing the previous body after the
			// response has arrived, so every iteration needs its own reader (2 allocs).
			request.Body = io.NopCloser(strings.NewReader(benchmarkPayload))
			comp.ServeHTTP(recorder, request)
		}

		checkStatus(b, recorder)
	})
}

// newBenchmarkForwarder starts a test backend and returns a forwarder routing to it, along
// with a request context that targets that backend.
func newBenchmarkForwarder(b *testing.B) (composer.FlowComponent, context.Context) {
	b.Helper()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		_, _ = io.Copy(io.Discard, req.Body)
		w.WriteHeader(http.StatusOK)
	}))
	b.Cleanup(srv.Close)

	addr, _ := srv.Listener.Addr().(*net.TCPAddr)
	backend := &config.RouterBackend{Name: "backend1", Host: addr.IP.String(), Port: addr.Port}

	comp, err := NewComponent(&Opts{Backends: []*config.RouterBackend{backend}})
	if err != nil {
		b.Fatalf("failed to create forwarder component: %v", err)
	}

	ctx := context.WithValue(context.Background(), composer.TargetContextKey, backend)
	return comp, logr.NewContext(ctx, logr.Discard())
}

// checkStatus fails the benchmark if the forwarded request did not succeed. The recorder
// keeps the first status code written, so checking once after the loop is enough.
func checkStatus(b *testing.B, recorder *httptest.ResponseRecorder) {
	b.Helper()

	if recorder.Code != http.StatusOK {
		b.Fatalf("unexpected response code: got %d, want %d: %s",
			recorder.Code, http.StatusOK, recorder.Body.String())
	}
}
