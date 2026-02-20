package oas

import (
	"net/http"
	"os"
	"strconv"

	"github.com/getkin/kin-openapi/openapi3"
	"github.com/go-logr/logr"
	"github.com/trebent/kerberos/internal/composer/custom"
	composertypes "github.com/trebent/kerberos/internal/composer/types"
	"github.com/trebent/kerberos/internal/config"
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
	//nolint:reassign // yolo
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

// GetMeta implements [types.FlowComponent].
func (v *validator) GetMeta() composertypes.FlowMeta {
	meta := composertypes.FlowMeta{
		Name:        "oas-validator",
		Description: "Validates incoming requests against OpenAPI specifications.",
		Data: map[string]string{
			"backend_count": strconv.Itoa(len(v.validators)),
			"order":         strconv.Itoa(v.cfg.Order),
		},
	}
	if v.next != nil {
		next := v.next.GetMeta()
		meta.Next = &next
	}
	return meta
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

	v.validators[m.Backend] = ValidationMiddleware(spec)

	return nil
}
