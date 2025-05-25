// nolint: mnd
package response

import (
	"context"
	"errors"
	"io"
	"net/http"
	"sync"

	"github.com/trebent/zerologr"
	"go.opentelemetry.io/otel/codes"
)

type (
	BodyWrapper struct {
		body  io.ReadCloser
		bytes int64
	}
	ResponseWrapper struct {
		responseWriter http.ResponseWriter
		lock           *sync.Mutex

		requestContext context.Context

		bytes       int64
		wroteHeader bool
		statusCode  int
	}
)

var (
	_ http.ResponseWriter = &ResponseWrapper{}
	_ http.Flusher        = &ResponseWrapper{}

	_ io.ReadCloser = &BodyWrapper{}
)

func NewBodyWrapper(body io.ReadCloser) io.ReadCloser {
	return &BodyWrapper{body: body}
}

func (bw *BodyWrapper) Close() error {
	return bw.body.Close()
}

func (bw *BodyWrapper) Read(p []byte) (int, error) {
	n, err := bw.body.Read(p)
	bw.bytes += int64(n)
	return n, err
}

func (bw *BodyWrapper) NumBytes() int64 {
	return bw.bytes
}

func UpdateRequestContext(w http.ResponseWriter, ctx context.Context) {
	if rw, ok := w.(*ResponseWrapper); ok {
		rw.SetRequestContext(ctx)
		return
	}
	zerologr.Error(errors.New("wrong type"), "UpdateRequestContext called with non-responseWrapper type")
}

func NewResponseWrapper(responseWriter http.ResponseWriter) http.ResponseWriter {
	return &ResponseWrapper{lock: &sync.Mutex{}, responseWriter: responseWriter}
}

func (r *ResponseWrapper) RealResponseWriter() http.ResponseWriter {
	return r.responseWriter
}

func (r *ResponseWrapper) SetRequestContext(ctx context.Context) {
	r.requestContext = ctx
}

func (r *ResponseWrapper) GetRequestContext() context.Context {
	if r.requestContext == nil {
		return context.Background()
	}
	return r.requestContext
}

func (r *ResponseWrapper) Header() http.Header {
	return r.responseWriter.Header()
}

func (r *ResponseWrapper) Write(p []byte) (int, error) {
	zerologr.V(100).Info("Write", "len", len(p))

	n, err := r.responseWriter.Write(p)
	r.bytes += int64(n)
	return n, err
}

func (r *ResponseWrapper) WriteHeader(statusCode int) {
	zerologr.V(100).Info("WriteHeader", "status_code", statusCode)

	r.lock.Lock()
	defer r.lock.Unlock()

	if !r.wroteHeader {
		r.wroteHeader = true
		r.statusCode = statusCode
	}

	r.responseWriter.WriteHeader(statusCode)
}

func (r *ResponseWrapper) Flush() {
	zerologr.V(100).Info("Flush response")

	r.WriteHeader(http.StatusOK)

	if f, ok := r.responseWriter.(http.Flusher); ok {
		f.Flush()
	}
}

func (r *ResponseWrapper) NumBytes() int64 {
	return r.bytes
}

func (r *ResponseWrapper) StatusCode() int {
	return r.statusCode
}

func (r *ResponseWrapper) SpanStatus() (codes.Code, string) {
	if !r.wroteHeader {
		return codes.Error, "no available status code"
	}

	if r.statusCode >= 400 {
		return codes.Error, http.StatusText(r.statusCode)
	}

	return codes.Ok, http.StatusText(r.statusCode)
}
