package forwarder

import (
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"strconv"

	"github.com/go-logr/logr"
	apierror "github.com/trebent/kerberos/internal/api/error"
	composertypes "github.com/trebent/kerberos/internal/composer/types"
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

	errFailedTargetExtract = errors.New("could not determine target from context")
	//nolint:errname // This is intentional to separate pure error types from wrapper API Errors.
	apiErrFailedTargetExtract = apierror.New(
		http.StatusInternalServerError,
		http.StatusText(http.StatusInternalServerError),
	)
	errFailedForwarding = errors.New("failed to forward request")
	//nolint:errname // This is intentional to separate pure error types from wrapper API Errors.
	apiErrFailedForwarding = apierror.New(
		http.StatusInternalServerError,
		errFailedForwarding.Error(),
	)
)

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
			fmt.Errorf("%w: %s", errFailedTargetExtract, req.URL.Path),
			"Target extract failed",
		)
		apierror.ErrorHandler(wrapped, req, apiErrFailedTargetExtract)
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
		apierror.ErrorHandler(wrapped, req, apiErrFailedForwarding)
		return
	}

	forwardRequest.Header = req.Header
	otel.GetTextMapPropagator().
		Inject(req.Context(), propagation.HeaderCarrier(forwardRequest.Header))

	client := http.Client{}
	resp, err := client.Do(forwardRequest)
	if err != nil {
		rLogger.Error(err, "Failed to forward request")
		apierror.ErrorHandler(wrapped, req, apiErrFailedForwarding)
		return
	}
	defer resp.Body.Close()

	for key, values := range resp.Header {
		rLogger.V(100).Info("Adding header to response", "key", key, "values", values)
		for _, value := range values {
			wrapped.Header().Add(key, value)
		}
	}

	wrapped.WriteHeader(resp.StatusCode)

	_, err = io.Copy(wrapped, resp.Body)
	if err != nil {
		rLogger.Error(err, "Failed to copy response body")
		apierror.ErrorHandler(wrapped, req, apiErrFailedForwarding)
		return
	}

	rLogger.V(50).Info("Forwarded request")
}
