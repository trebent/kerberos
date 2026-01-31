package oas

import (
	"context"
	"net/http"
	"os"

	"github.com/getkin/kin-openapi/openapi3"
	"github.com/go-logr/logr"
	nethttpmiddleware "github.com/oapi-codegen/nethttp-middleware"
	"github.com/trebent/kerberos/internal/composer/custom"
	composertypes "github.com/trebent/kerberos/internal/composer/types"
	"github.com/trebent/kerberos/internal/config"
	"github.com/trebent/kerberos/internal/response"
	"github.com/trebent/zerologr"
)

type (
	validator struct {
		next composertypes.FlowComponent
		// Map of backend name to OAS validator handler.
		validators map[string]func(http.Handler) http.Handler
		cfg        *oasConfig
	}
	Opts struct {
		Cfg config.Map

		// TODO: use this to register API documentation.
		Mux *http.ServeMux
	}
)

var (
	_ composertypes.FlowComponent = &validator{}
	_ custom.Ordered              = &validator{}
)

func New(opts *Opts) composertypes.FlowComponent {
	// Prevent schema error details from being included in validation errors.
	openapi3.SchemaErrorDetailsDisabled = true

	cfg := config.AccessAs[*oasConfig](opts.Cfg, configName)
	v := &validator{cfg: cfg, validators: make(map[string]func(http.Handler) http.Handler)}

	for _, mapping := range cfg.Mappings {
		if err := v.register(mapping); err != nil {
			panic(err)
		}
	}

	return v
}

func (v *validator) Order() int {
	return v.cfg.Order
}

// Next implements [types.FlowComponent].
func (v *validator) Next(next composertypes.FlowComponent) {
	v.next = next
}

// ServeHTTP implements [types.FlowComponent].
func (v *validator) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	backend, _ := req.Context().Value(composertypes.BackendContextKey).(string)
	oLogger, _ := logr.FromContext(req.Context())
	oLogger = oLogger.WithName("oas-validator")
	oLogger.Info("Running OAS validation", "backend", backend)

	if handler, ok := v.validators[backend]; ok {
		handler(v.next).ServeHTTP(w, req)
	} else {
		v.next.ServeHTTP(w, req)
	}
}

func (v *validator) register(m *mapping) error {
	if m.Options == nil {
		m.Options = defaultOptions()
	}

	// Load OpenAPI document.
	zerologr.Info("Preparing OAS validator", "backend", m.Backend)
	bs, err := os.ReadFile(m.Specification)
	if err != nil {
		return err
	}

	spec, err := openapi3.NewLoader().LoadFromData(bs)
	if err != nil {
		return err
	}

	if !m.Options.ValidateBody {
		zerologr.Info("Disabling body validation", "backend", m.Backend)
		for _, path := range spec.Paths.Map() {
			for _, op := range path.Operations() {
				op.RequestBody = nil
			}
		}
	}

	opts := &nethttpmiddleware.Options{
		SilenceServersWarning: true,
		DoNotValidateServers:  true,
		ErrorHandlerWithOpts:  v.oasValidationErrorHandler,
	}

	v.validators[m.Backend] = nethttpmiddleware.OapiRequestValidatorWithOptions(spec, opts)

	return nil
}

func (v *validator) oasValidationErrorHandler(
	ctx context.Context,
	err error,
	w http.ResponseWriter,
	req *http.Request,
	opts nethttpmiddleware.ErrorHandlerOpts,
) {
	logger, _ := logr.FromContext(ctx)
	logger = logger.WithName("oas-validator")
	logger.Error(err, "OAS validation failed",
		"backend", req.Context().Value(composertypes.BackendContextKey),
		"path", req.URL.Path,
	)
	response.JSONError(w, err, opts.StatusCode)
}
