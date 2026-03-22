// nolint: mnd
package response

import (
	"context"
	"io"
	"net/http"

	"github.com/trebent/zerologr"
	"go.opentelemetry.io/otel/codes"
)

type (
	BodyWrapper struct {
		body  io.ReadCloser
		bytes int64
	}
	Wrapper struct {
		responseWriter http.ResponseWriter

		requestContext context.Context

		bytes       int64
		wroteHeader bool
		statusCode  int
	}
)

var (
	_ http.ResponseWriter = &Wrapper{}
	_ http.Flusher        = &Wrapper{}

	_ io.ReadCloser = &BodyWrapper{}
)

// NewBodyWrapper creates a new BodyWrapper that wraps the provided io.ReadCloser.
// When the body is Read from, it counts the number of bytes read. When Close is called,
// it closes the underlying body.
func NewBodyWrapper(body io.ReadCloser) io.ReadCloser {
	return &BodyWrapper{body: body}
}

// Close the underlying body and returns any error encountered during closing.
func (bw *BodyWrapper) Close() error {
	return bw.body.Close()
}

// Read reads data from the underlying body into the provided byte slice. It counts the
// number of bytes read and returns the count along with any error encountered during reading.
func (bw *BodyWrapper) Read(p []byte) (int, error) {
	n, err := bw.body.Read(p)
	bw.bytes += int64(n)
	return n, err
}

// NumBytes returns the total number of bytes read from the body.
func (bw *BodyWrapper) NumBytes() int64 {
	return bw.bytes
}

// NewResponseWrapper creates a new Wrapper that wraps the provided http.ResponseWriter. The Wrapper
// allows for tracking the number of bytes written, the status code, and provides access to the request
// context. It also implements the http.Flusher interface to allow for flushing the response when needed.
func NewResponseWrapper(responseWriter http.ResponseWriter) http.ResponseWriter {
	return &Wrapper{responseWriter: responseWriter}
}

// SetRequestContext sets the request context for the Wrapper. This context can be used to store and retrieve
// information related to the request being processed, such as tracing information or other metadata.
func (r *Wrapper) SetRequestContext(ctx context.Context) {
	r.requestContext = ctx
}

// GetRequestContext retrieves the request context associated with the Wrapper. If no context has been set,
// it returns a background context.
func (r *Wrapper) GetRequestContext() context.Context {
	if r.requestContext == nil {
		return context.Background()
	}
	return r.requestContext
}

// Header returns the header map that will be sent by WriteHeader. The Header map also is the mechanism with
// which Handlers can set HTTP trailers. By default, the returned map is initialized to be empty, and
// HTTP handlers can add key-value pairs to it as needed. The Header map is not thread-safe, so it should
// only be modified by the handler that is processing the request.
func (r *Wrapper) Header() http.Header {
	return r.responseWriter.Header()
}

// Write writes the data to the connection as part of an HTTP reply. It counts the number of bytes written and
// returns the count along with any error encountered during writing. If WriteHeader has not yet been called,
// Write calls WriteHeader(http.StatusOK) before writing the data.
func (r *Wrapper) Write(p []byte) (int, error) {
	zerologr.V(100).Info("Write", "len", len(p))

	if !r.wroteHeader {
		r.WriteHeader(http.StatusOK)
	}

	n, err := r.responseWriter.Write(p)
	r.bytes += int64(n)
	return n, err
}

// WriteHeader sends an HTTP response header with the provided status code. If WriteHeader is called multiple times,
// only the first call will have an effect, and subsequent calls will be ignored. The status code is stored
// in the Wrapper for later retrieval, and the header is sent to the client using the underlying http.ResponseWriter.
func (r *Wrapper) WriteHeader(statusCode int) {
	zerologr.V(100).Info("WriteHeader", "status_code", statusCode)

	if !r.wroteHeader {
		r.wroteHeader = true
		r.statusCode = statusCode
		r.responseWriter.WriteHeader(statusCode)
	}
}

func (r *Wrapper) Flush() {
	zerologr.V(100).Info("Flush response")

	r.WriteHeader(http.StatusOK)

	if f, ok := r.responseWriter.(http.Flusher); ok {
		f.Flush()
	}
}

// NumBytes returns the total number of bytes written to the response.
func (r *Wrapper) NumBytes() int64 {
	return r.bytes
}

// StatusCode returns the HTTP status code that was sent in the response. If WriteHeader has not been called, it returns 0.
func (r *Wrapper) StatusCode() int {
	return r.statusCode
}

// SpanStatus returns the OpenTelemetry span status code and description based on the HTTP status code of the response.
// If WriteHeader has not been called, it returns codes.Error with a description indicating that no status code is available.
// If the HTTP status code is 400 or higher, it returns codes.Error with the corresponding HTTP status text as the description.
// For status codes below 400, it returns codes.Ok with the corresponding HTTP status text as the description.
func (r *Wrapper) SpanStatus() (codes.Code, string) {
	if !r.wroteHeader {
		return codes.Error, "no available status code"
	}

	if r.statusCode >= 400 {
		return codes.Error, http.StatusText(r.statusCode)
	}

	return codes.Ok, http.StatusText(r.statusCode)
}
