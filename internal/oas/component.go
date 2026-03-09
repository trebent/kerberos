package oas

import (
	"net/http"
	"os"

	"github.com/getkin/kin-openapi/openapi3"
	"github.com/go-logr/logr"
	"github.com/trebent/kerberos/internal/composer"
	"github.com/trebent/kerberos/internal/composer/custom"
	"github.com/trebent/kerberos/internal/config"
	"github.com/trebent/zerologr"
)

type (
	validator struct {
		next composer.FlowComponent
		// Map of backend name to OAS validator handler.
		validators map[string]func(http.Handler) http.Handler
		cfg        *config.OASConfig
	}
	Opts struct {
		Cfg *config.OASConfig

		// TODO: use this to register API documentation.
		Mux *http.ServeMux
	}
)

var (
	_ composer.FlowComponent = &validator{}
	_ custom.Ordered         = &validator{}
)

func NewComponent(opts *Opts) composer.FlowComponent {
	// Prevent schema error details from being included in validation errors.
	//nolint:reassign // yolo
	openapi3.SchemaErrorDetailsDisabled = true

	v := &validator{cfg: opts.Cfg, validators: make(map[string]func(http.Handler) http.Handler)}
	for _, mapping := range v.cfg.Mappings {
		if err := v.register(&mapping); err != nil {
			panic(err)
		}
	}

	return v
}

func (v *validator) Order() int {
	return v.cfg.Order
}

// Next implements [composer.FlowComponent].
func (v *validator) Next(next composer.FlowComponent) {
	v.next = next
}

// GetMeta implements [composer.FlowComponent].
func (v *validator) GetMeta() []*composer.FlowMeta {
	return append([]*composer.FlowMeta{
		{
			Name: "oas-validator",
			Data: map[string]any{
				"backends": func() []string {
					backends := make([]string, 0, len(v.validators))
					for backend := range v.validators {
						backends = append(backends, backend)
					}
					return backends
				}(),
				composer.MetaKeyOrder: v.cfg.Order,
			},
		},
	}, v.next.GetMeta()...)
}

// ServeHTTP implements [composer.FlowComponent].
func (v *validator) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	backend, _ := req.Context().Value(composer.BackendContextKey).(string)
	oLogger, _ := logr.FromContext(req.Context())
	oLogger = oLogger.WithName("oas-validator")
	oLogger.Info("Running OAS validation", "backend", backend)

	if handler, ok := v.validators[backend]; ok {
		handler(v.next).ServeHTTP(w, req)
	} else {
		v.next.ServeHTTP(w, req)
	}
}

func (v *validator) register(m *config.OASBackendMapping) error {
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
