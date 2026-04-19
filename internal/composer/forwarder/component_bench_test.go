package forwarder

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"

	"github.com/go-logr/logr"
	"github.com/trebent/kerberos/internal/composer"
	"github.com/trebent/kerberos/internal/config"
	"github.com/trebent/zerologr"
)

func BenchmarkForwarder_SingleBackend_NoBody_NoTLS(b *testing.B) {
	testSrv := httptest.NewUnstartedServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	testSrv.Start() // Start without TLS since we're testing the non-TLS code path of the forwarder.
	defer testSrv.Close()

	split := strings.Split(testSrv.URL, "://")
	hostPort := strings.Split(split[1], ":")
	b.Log(hostPort)

	port, err := strconv.Atoi(hostPort[1])
	if err != nil {
		b.Fatalf("failed to parse test server port: %v", err)
	}

	backend := &config.RouterBackend{
		Name: "backend1",
		Host: hostPort[0],
		Port: port,
	}

	comp, err := NewComponent(&Opts{
		Backends: []*config.RouterBackend{
			backend,
		},
	})
	if err != nil {
		b.Fatalf("failed to create forwarder component: %v", err)
	}

	reqCtx := context.WithValue(context.Background(), composer.TargetContextKey, backend)
	reqCtx = logr.NewContext(reqCtx, zerologr.New(&zerologr.Opts{Console: true}))
	b.ResetTimer()

	for b.Loop() {
		recorder := httptest.NewRecorder()

		req := httptest.NewRequest(http.MethodGet, "/", nil).WithContext(reqCtx)

		comp.ServeHTTP(recorder, req)
		if recorder.Code != http.StatusOK {
			b.Error(recorder.Body.String())

			b.Fatalf("unexpected response code: got %d, want %d", recorder.Code, http.StatusOK)
		}
	}
}

func BenchmarkForwarder_SingleBackend_Body_NoTLS(b *testing.B) {
	testSrv := httptest.NewUnstartedServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	testSrv.Start() // Start without TLS since we're testing the non-TLS code path of the forwarder.
	defer testSrv.Close()

	split := strings.Split(testSrv.URL, "://")
	hostPort := strings.Split(split[1], ":")
	b.Log(hostPort)

	port, err := strconv.Atoi(hostPort[1])
	if err != nil {
		b.Fatalf("failed to parse test server port: %v", err)
	}

	backend := &config.RouterBackend{
		Name: "backend1",
		Host: hostPort[0],
		Port: port,
	}

	comp, err := NewComponent(&Opts{
		Backends: []*config.RouterBackend{
			backend,
		},
	})
	if err != nil {
		b.Fatalf("failed to create forwarder component: %v", err)
	}

	reqCtx := context.WithValue(context.Background(), composer.TargetContextKey, backend)
	reqCtx = logr.NewContext(reqCtx, zerologr.New(&zerologr.Opts{Console: true}))
	b.ResetTimer()

	for b.Loop() {
		recorder := httptest.NewRecorder()

		req := httptest.NewRequest(
			http.MethodGet,
			"/",
			bytes.NewBuffer([]byte("somebytessomebytessomebytessomebytessomebytessomebytessomebytessomebytessomebytessomebytessomebytessomebytessomebytessomebytessomebytessomebytes")),
		).WithContext(reqCtx)

		comp.ServeHTTP(recorder, req)
		if recorder.Code != http.StatusOK {
			b.Error(recorder.Body.String())

			b.Fatalf("unexpected response code: got %d, want %d", recorder.Code, http.StatusOK)
		}
	}
}
