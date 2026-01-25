package forwarder

import (
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"strconv"

	"github.com/go-logr/logr"
	composertypes "github.com/trebent/kerberos/internal/composer/types"
	"github.com/trebent/kerberos/internal/response"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/propagation"
)

type (
	Target interface {
		Host() string
		Port() int
	}
	Opts struct {
		TargetContextKey composertypes.ContextKey
	}
	forwarder struct {
		targetContextKey composertypes.ContextKey
	}
)

var (
	_ composertypes.FlowComponent = (*forwarder)(nil)

	ErrFailedPatternMatch  = errors.New("forward pattern match failed")
	ErrFailedTargetExtract = errors.New("could not determine target from context")
	ErrFailedForwarding    = errors.New("failed to forward request")
)

const expectedPatternMatches = 2

func NewComponent(opts *Opts) composertypes.FlowComponent {
	return &forwarder{
		targetContextKey: opts.TargetContextKey,
	}
}

// Next implements [types.FlowComponent].
func (f *forwarder) Next(_ composertypes.FlowComponent) {
	panic("the forwarder is intended to be the last component in the flow")
}

// ServeHTTP implements [types.FlowComponent].
func (f *forwarder) ServeHTTP(wrapped http.ResponseWriter, req *http.Request) {
	// Obtain matching backend to route to.
	// Forward request and pipe forwarded response into origin response.
	rLogger, _ := logr.FromContext(req.Context())
	rLogger = rLogger.WithName("forwarder")
	rLogger.Info("Forwarding request")

	target, ok := req.Context().Value(f.targetContextKey).(Target)
	if !ok {
		rLogger.Error(
			fmt.Errorf("%w: %s", ErrFailedTargetExtract, req.URL.Path),
			"Target extract failed",
		)
		response.JSONError(wrapped, ErrFailedForwarding, http.StatusInternalServerError)
		return
	}

	forwardRequest, err := http.NewRequestWithContext(
		req.Context(),
		req.Method,
		fmt.Sprintf(
			"http://%s%s",
			net.JoinHostPort(target.Host(), strconv.Itoa(target.Port())),
			req.URL.Path,
		),
		req.Body,
	)
	if err != nil {
		rLogger.Error(err, "Failed to create request")
		response.JSONError(wrapped, ErrFailedForwarding, http.StatusInternalServerError)
		return
	}

	forwardRequest.Header = req.Header
	otel.GetTextMapPropagator().
		Inject(req.Context(), propagation.HeaderCarrier(forwardRequest.Header))

	client := http.Client{}
	resp, err := client.Do(forwardRequest)
	if err != nil {
		rLogger.Error(err, "Failed to forward request")
		response.JSONError(wrapped, ErrFailedForwarding, http.StatusInternalServerError)
		return
	}
	defer resp.Body.Close()

	for key, values := range resp.Header {
		rLogger.V(100).Info("Adding header to response", "key", key, "values", values)
		for _, value := range values {
			wrapped.Header().Add(key, value)
		}
	}

	_, err = io.Copy(wrapped, resp.Body)
	if err != nil {
		rLogger.Error(err, "Failed to copy response body")
		response.JSONError(wrapped, ErrFailedForwarding, http.StatusInternalServerError)
		return
	}

	wrapped.WriteHeader(resp.StatusCode)
	rLogger.V(50).Info("Forwarded request")
}
