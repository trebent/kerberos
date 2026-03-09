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
	"github.com/trebent/kerberos/internal/composer"
	"github.com/trebent/kerberos/internal/config"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/propagation"
)

type (
	// Placeholder options struct for future use, currently empty.
	Opts      struct{}
	forwarder struct {
		targetContextKey composer.ContextKey
		client           *http.Client
	}
)

var (
	_ composer.FlowComponent = (*forwarder)(nil)

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

func NewComponent(_ *Opts) composer.FlowComponent {
	return &forwarder{
		targetContextKey: composer.TargetContextKey,
		// TODO: determine timeouts from input configuration
		client: &http.Client{},
	}
}

// Next implements [composer.FlowComponent].
func (f *forwarder) Next(_ composer.FlowComponent) {
	panic("the forwarder is intended to be the last component in the flow")
}

// GetMeta implements [composer.FlowComponent].
func (f *forwarder) GetMeta() []*composer.FlowMeta {
	return []*composer.FlowMeta{
		{
			Name: "forwarder",
			Data: map[string]any{},
		},
	}
}

// ServeHTTP implements [composer.FlowComponent].
func (f *forwarder) ServeHTTP(wrapped http.ResponseWriter, req *http.Request) {
	// Obtain matching backend to route to.
	// Forward request and pipe forwarded response into origin response.
	rLogger, _ := logr.FromContext(req.Context())
	rLogger = rLogger.WithName("forwarder")
	rLogger.Info("Forwarding request")

	target, ok := req.Context().Value(f.targetContextKey).(*config.RouterBackend)
	if !ok {
		rLogger.Error(
			fmt.Errorf("%w: %s", errFailedTargetExtract, req.URL.Path),
			"Target extract failed",
		)
		apierror.ErrorHandler(wrapped, req, apiErrFailedTargetExtract)
		return
	}

	//nolint:gosec // ignoring SSRF warning since the target is determined by our own routing logic and not user input.
	forwardRequest, err := http.NewRequestWithContext(
		req.Context(),
		req.Method,
		fmt.Sprintf(
			"http://%s%s",
			net.JoinHostPort(target.Host, strconv.Itoa(target.Port)),
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

	//nolint:gosec // ignoring SSRF warning since the target is determined by our own routing logic and not user input.
	resp, err := f.client.Do(forwardRequest)
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
