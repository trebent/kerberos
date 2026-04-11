package config

type (
	// OASConfig holds configuration for OAS-based request routing and validation.
	OASConfig struct {
		Order    int                  `json:"order"`
		Mappings []*OASBackendMapping `json:"mappings"`
	}
	OASBackendMapping struct {
		Backend       string                 `json:"backend"`
		Specification string                 `json:"specification"`
		Options       *OASBackendMappingOpts `json:"options"`
	}
	OASBackendMappingOpts struct {
		ValidateBody bool `json:"validateBody"`
	}

	// RouterConfig holds configuration for the request router.
	RouterConfig struct {
		Backends []*RouterBackend `json:"backends"`
	}
	RouterBackend struct {
		Name      string      `json:"name"`
		Host      string      `json:"host"`
		Port      int         `json:"port"`
		TimeoutMs int         `json:"timeout,omitempty"`
		TLS       *BackendTLS `json:"tls,omitempty"`
	}
	// BackendTLS holds per-backend TLS settings.
	// When nil, the forwarder uses plain HTTP for that backend.
	BackendTLS struct {
		// RootCAFile is the path to a PEM-encoded CA bundle used to verify the backend's certificate.
		// When empty, the system certificate pool is used.
		RootCAFile string `json:"rootCAFile,omitempty"`
		// ClientCertFile and ClientKeyFile enable mTLS.
		// Both must be set together.
		ClientCertFile string `json:"clientCertFile,omitempty"`
		ClientKeyFile  string `json:"clientKeyFile,omitempty"`
		// InsecureSkipVerify disables server certificate verification.
		// Must only be used in non-production environments.
		InsecureSkipVerify bool `json:"insecureSkipVerify,omitempty"`
	}

	// ObservabilityConfig holds configuration for observability features.
	ObservabilityConfig struct {
		Enabled        bool `json:"enabled"`
		RuntimeMetrics bool `json:"runtimeMetrics"`
	}

	// AuthConfig holds configuration for authentication and authorization.
	AuthConfig struct {
		Methods *AuthMethods `json:"methods"`
		Scheme  *AuthScheme  `json:"scheme"`
		Order   int          `json:"order"`
	}
	AuthMethods struct {
		Basic *AuthMethodBasic `json:"basic"`
	}
	AuthScheme struct {
		Mappings []*AuthMapping `json:"mappings"`
	}
	AuthMapping struct {
		Backend       string   `json:"backend"`
		Method        string   `json:"method"`
		Exempt        []string `json:"exempt"`
		Authorization *AuthZ   `json:"authorization"`
	}
	AuthZ struct {
		Groups []string            `json:"groups"`
		Paths  map[string][]string `json:"paths"`
	}
	AuthMethodBasic struct{}

	// AdminConfig holds configuration for the admin API.
	AdminConfig struct {
		SuperUser *SuperUser `json:"superUser"`
	}
	SuperUser struct {
		ClientID     string `json:"clientId"`
		ClientSecret string `json:"clientSecret"`
	}
)

func newAdminConfig() *AdminConfig {
	return &AdminConfig{
		SuperUser: &SuperUser{
			ClientID:     "admin",
			ClientSecret: "secret",
		},
	}
}

func newObservabilityConfig() *ObservabilityConfig {
	return &ObservabilityConfig{
		Enabled:        true,
		RuntimeMetrics: true,
	}
}

func (ac *AuthConfig) postProcess() {}
func (rc *RouterConfig) postProcess() {
	for _, b := range rc.Backends {
		if b.TimeoutMs == 0 {
			b.TimeoutMs = 5000 // default timeout of 5000 milliseconds
		}
	}
}
func (oc *ObservabilityConfig) postProcess() {}
func (ac *AdminConfig) postProcess()         {}
func (oc *OASConfig) postProcess() {
	for _, m := range oc.Mappings {
		if m.Options == nil {
			m.Options = &OASBackendMappingOpts{ValidateBody: true}
		}
	}
}
