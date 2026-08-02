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

	// GatewayConfig holds configuration for the API gateway.
	GatewayConfig struct {
		Router *Router    `json:"router"`
		TLS    *ServerTLS `json:"tls,omitempty"`
	}

	// Router holds configuration for the request router.
	Router struct {
		Backends []*RouterBackend `json:"backends"`
	}
	RouterBackend struct {
		Name      string `json:"name"`
		Host      string `json:"host"`
		Port      int    `json:"port"`
		TimeoutMs int    `json:"timeout,omitempty"`
		// Origins holds configuration for CORS origins. In addition, Origins other than the allowed ones
		// will be rejected with a 403 response.
		Origins *Origins    `json:"origins,omitempty"`
		TLS     *BackendTLS `json:"tls,omitempty"`
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
		API       *AdminAPI  `json:"api,omitempty"`
	}
	SuperUser struct {
		ClientID     string `json:"clientId"`
		ClientSecret string `json:"clientSecret"`
	}
	// AdminAPI holds configuration for the admin API.
	AdminAPI struct {
		// Cookies contain cookie settings for the csrf, session, and refresh cookies.
		Cookies *Cookies `json:"cookies,omitempty"`
		// Origins holds configuration for CORS origins. In addition, Origins other than the allowed ones
		// will be rejected with a 403 response.
		Origins *Origins `json:"origins,omitempty"`
		// TLS holds configuration for TLS settings for the admin API.
		TLS *ServerTLS `json:"tls,omitempty"`
	}

	Cookies struct {
		// Domain is the domain setting for cookies, this translates directly to Domain=<value> for cookies.
		Domain string `json:"domain,omitempty"`
		// SameSite is the SameSite setting for cookies, this translates directly to SameSite=<value> for cookies.
		SameSite string `json:"sameSite,omitempty"`
	}

	// Origins holds configuration for CORS origins.
	Origins struct {
		// AllowedOrigins is a list of allowed origins for CORS.
		AllowedOrigins []string `json:"allowedOrigins,omitempty"`
		// AllowAll indicates whether to allow all origins for CORS. Mutually exclusive with 'allowedOrigins'.
		// AllowAll will mean the Access-Control-Allow-Origin header is set to whatever Origin was received.
		AllowAll bool `json:"allowAll,omitempty"`
		// DenyAll denies any request with an Origin header, effectively disabling browser access.
		// Mutually exclusive with 'allowedOrigins' and 'allowAll'.
		DenyAll bool `json:"denyAll,omitempty"`
	}

	ServerTLS struct {
		CertFile string `json:"serverCertFile"`
		KeyFile  string `json:"serverKeyFile"`
	}

	// PersistenceConfig holds configuration for the backing database.
	PersistenceConfig struct {
		// Driver selects the database backend: "sqlite" or "postgres".
		Driver string `json:"driver"`
		// Address is the database address. For postgres: host. For sqlite: file path.
		Address string `json:"address"`

		// Postgres contains specific configuration for the postgres driver. Ignored for other drivers.
		*Postgres `json:"postgres,omitempty"`
	}
	Postgres struct {
		// Database is the database name (postgres only).
		Database string `json:"database"`
		// Username is the database user (postgres only).
		Username *string `json:"username,omitempty"`
		// Password is the database password (postgres only).
		Password *string `json:"password,omitempty"`
		// SSLMode controls TLS for postgres connections (e.g. "disable", "require", "verify-full").
		SSLMode *string `json:"sslMode,omitempty"`
	}
)

const defaultCalloutTimeoutMs = 5000

func newAdminConfig() *AdminConfig {
	return &AdminConfig{
		API: &AdminAPI{
			Cookies: &Cookies{
				SameSite: "Strict",
			},
			// Default as empty to simplify boot configuration, normally this will fail validation
			// as both allow all and allowed origins are empty, but this is a valid default for bootstrapping.
			Origins: &Origins{},
		},
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

func newPersistenceConfig() *PersistenceConfig {
	return &PersistenceConfig{
		Driver:  "sqlite",
		Address: "krb.db",
	}
}

func (ac *AuthConfig) postProcess() {}
func (gc *GatewayConfig) postProcess() {
	for _, b := range gc.Router.Backends {
		if b.TimeoutMs == 0 {
			b.TimeoutMs = defaultCalloutTimeoutMs
		}
	}
}
func (pc *PersistenceConfig) postProcess()   {}
func (oc *ObservabilityConfig) postProcess() {}
func (ac *AdminConfig) postProcess()         {}
func (oc *OASConfig) postProcess() {
	for _, m := range oc.Mappings {
		if m.Options == nil {
			m.Options = &OASBackendMappingOpts{ValidateBody: true}
		}
	}
}
