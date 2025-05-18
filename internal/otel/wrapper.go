// nolint: mnd
package otel

import (
	"io"
	"net/http"
	"sync"

	"github.com/felixge/httpsnoop"
	"github.com/trebent/zerologr"
	"go.opentelemetry.io/otel/codes"
)

type (
	bodyWrapper struct {
		body  io.ReadCloser
		bytes int64
	}
	responseWrapper struct {
		responseWriter http.ResponseWriter
		lock           *sync.Mutex

		bytes       int64
		wroteHeader bool
		statusCode  int
	}
)

var (
	_ http.ResponseWriter = &responseWrapper{}
	_ http.Flusher        = &responseWrapper{}
	_ io.ReadCloser       = &bodyWrapper{}
)

func newBodyWrapper(body io.ReadCloser) io.ReadCloser {
	return &bodyWrapper{body: body}
}

func (bw *bodyWrapper) Close() error {
	return bw.body.Close()
}

func (bw *bodyWrapper) Read(p []byte) (int, error) {
	n, err := bw.body.Read(p)
	bw.bytes += int64(n)
	return n, err
}

func (bw *bodyWrapper) NumBytes() int64 {
	return bw.bytes
}

func newResponseWrapper(
	responseWriter http.ResponseWriter,
) (http.ResponseWriter, *responseWrapper) {
	rw := &responseWrapper{responseWriter: responseWriter, lock: &sync.Mutex{}}
	return httpsnoop.Wrap(responseWriter, httpsnoop.Hooks{
		Header: func(httpsnoop.HeaderFunc) httpsnoop.HeaderFunc {
			return rw.Header
		},
		Write: func(httpsnoop.WriteFunc) httpsnoop.WriteFunc {
			return rw.Write
		},
		WriteHeader: func(httpsnoop.WriteHeaderFunc) httpsnoop.WriteHeaderFunc {
			return rw.WriteHeader
		},
		Flush: func(httpsnoop.FlushFunc) httpsnoop.FlushFunc {
			return rw.Flush
		},
	}), rw
}

func (r *responseWrapper) Header() http.Header {
	zerologr.V(100).Info("Header")

	return r.responseWriter.Header()
}

func (r *responseWrapper) Write(p []byte) (int, error) {
	zerologr.V(100).Info("Write", "len", len(p))

	n, err := r.responseWriter.Write(p)
	r.bytes += int64(n)
	return n, err
}

func (r *responseWrapper) WriteHeader(statusCode int) {
	zerologr.V(100).Info("WriteHeader", "status_code", statusCode)

	r.lock.Lock()
	defer r.lock.Unlock()

	if !r.wroteHeader {
		r.wroteHeader = true
		r.statusCode = statusCode
	}

	r.responseWriter.WriteHeader(statusCode)
}

func (r *responseWrapper) Flush() {
	zerologr.V(100).Info("Flush response")

	r.WriteHeader(http.StatusOK)

	if f, ok := r.responseWriter.(http.Flusher); ok {
		f.Flush()
	}
}

func (r *responseWrapper) NumBytes() int64 {
	return r.bytes
}

func (r *responseWrapper) StatusCode() int {
	return r.statusCode
}

func (r *responseWrapper) SpanStatus() (codes.Code, string) {
	if !r.wroteHeader {
		return codes.Error, "no available status code"
	}

	if r.statusCode >= 400 {
		return codes.Error, http.StatusText(r.statusCode)
	}

	return codes.Ok, http.StatusText(r.statusCode)
}
