package forwarder

import (
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"strconv"
	"time"

	"github.com/go-logr/logr"
	"github.com/trebent/kerberos/internal/composer"
	composerdebug "github.com/trebent/kerberos/internal/composer/debug"
	"github.com/trebent/kerberos/internal/config"
	adminapi "github.com/trebent/kerberos/internal/oapi/admin"
	apierror "github.com/trebent/kerberos/internal/oapi/error"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/propagation"
)

type (
	Opts struct {
		Backends []*config.RouterBackend
	}
	forwarder struct {
		targetContextKey composer.ContextKey
		clients          map[string]*http.Client // keyed by RouterBackend.Name
	}
)

var (
	_ composer.FlowComponent = (*forwarder)(nil)

	errFailedTargetExtract = errors.New("could not determine target from context")
	errFailedForwarding    = errors.New("failed to forward request")
	//nolint:errname // This is intentional to separate pure error types from wrapper API Errors.
	apiErrFailedForwarding = apierror.New(
		http.StatusInternalServerError,
		errFailedForwarding.Error(),
	)
)

func NewComponent(opts *Opts) (composer.FlowComponent, error) {
	clients := make(map[string]*http.Client, len(opts.Backends))
	for _, b := range opts.Backends {
		t, err := newTransport(b.Name, b.TLS)
		if err != nil {
			return nil, fmt.Errorf("building transport for backend %q: %w", b.Name, err)
		}

		clients[b.Name] = &http.Client{
			Transport: t,
			Timeout:   time.Duration(b.TimeoutMs) * time.Millisecond,
		}
	}
	return &forwarder{
		targetContextKey: composer.TargetContextKey,
		clients:          clients,
	}, nil
}

// Next implements [composer.FlowComponent].
func (f *forwarder) Next(_ composer.FlowComponent) {
	panic("the forwarder is intended to be the last component in the flow")
}

// GetMeta implements [composer.FlowComponent].
func (f *forwarder) GetMeta() []adminapi.FlowMeta {
	fmd := adminapi.FlowMeta_Data{}
	if err := fmd.FromNoFlowMetaData(adminapi.NoFlowMetaData{}); err != nil {
		panic(err)
	}

	return []adminapi.FlowMeta{
		{
			Name: "forwarder",
			Data: fmd,
		},
	}
}

// ServeHTTP implements [composer.FlowComponent].
func (f *forwarder) ServeHTTP(wrapped http.ResponseWriter, req *http.Request) {
	debugStart := time.Now()
	debugCall := composer.DebugFromContext(req.Context())

	// Obtain matching backend to route to.
	// Forward request and pipe forwarded response into origin response.
	rLogger, _ := logr.FromContext(req.Context())
	rLogger = rLogger.WithName("forwarder")
	rLogger.V(20).Info("Forwarding request")

	resp, err := f.handleInbound(req)
	if err != nil {
		rLogger.Error(err, "Failed to forward request")
		apierror.ErrorHandler(wrapped, req, apiErrFailedForwarding)
		debugCall.AddTransition(
			"forwarder",
			composerdebug.CallDirectionInbound,
			debugStart,
			time.Now(),
			composerdebug.CallResultFailure,
			err.Error(),
		)
		return
	}
	defer resp.Body.Close()

	debugCall.AddTransition(
		"forwarder",
		composerdebug.CallDirectionInbound,
		debugStart,
		time.Now(),
		composerdebug.CallResultSuccess,
		"",
	)

	// Reset to track outbound transition time.
	debugStart = time.Now()

	if err := f.handleOutbound(resp, wrapped, rLogger); err != nil {
		rLogger.Error(err, "Failed to handle outbound response")
		apierror.ErrorHandler(wrapped, req, apiErrFailedForwarding)
		debugCall.AddTransition(
			"forwarder",
			composerdebug.CallDirectionOutbound,
			debugStart,
			time.Now(),
			composerdebug.CallResultFailure,
			err.Error(),
		)
		return
	}

	debugCall.AddTransition(
		"forwarder",
		composerdebug.CallDirectionOutbound,
		debugStart,
		time.Now(),
		composerdebug.CallResultSuccess,
		"",
	)

	rLogger.V(50).Info("Forwarded request")
}

func (f *forwarder) handleInbound(req *http.Request) (*http.Response, error) {
	target, ok := req.Context().Value(f.targetContextKey).(*config.RouterBackend)
	if !ok {
		return nil, fmt.Errorf("%w: no target for: %s", errFailedTargetExtract, req.URL.Path)
	}

	client, ok := f.clients[target.Name]
	if !ok {
		return nil, fmt.Errorf("%w: no client for: %s", errFailedTargetExtract, req.URL.Path)
	}

	scheme := "http"
	if target.TLS != nil {
		scheme = "https"
	}

	//nolint:gosec // ignoring SSRF warning since the target is determined by our own routing logic and not user input.
	forwardRequest, err := http.NewRequestWithContext(
		req.Context(),
		req.Method,
		fmt.Sprintf(
			"%s://%s%s",
			scheme,
			net.JoinHostPort(target.Host, strconv.Itoa(target.Port)),
			req.URL.Path,
		),
		req.Body,
	)
	if err != nil {
		return nil, err
	}

	forwardRequest.Header = req.Header
	otel.GetTextMapPropagator().
		Inject(req.Context(), propagation.HeaderCarrier(forwardRequest.Header))

	//nolint:gosec // ignoring SSRF warning since the target is determined by our own
	// routing logic and not user input.
	return client.Do(forwardRequest)
}

func (f *forwarder) handleOutbound(
	resp *http.Response,
	wrapped http.ResponseWriter,
	rLogger logr.Logger,
) error {
	for key, values := range resp.Header {
		rLogger.V(100).Info("Adding header to response", "key", key, "values", values)
		for _, value := range values {
			wrapped.Header().Add(key, value)
		}
	}

	wrapped.WriteHeader(resp.StatusCode)

	_, err := io.Copy(wrapped, resp.Body)
	return err
}
