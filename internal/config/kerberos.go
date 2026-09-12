package config

import (
	_ "embed"

	utilhttp "github.com/trebent/kerberos/internal/util/http"
	"github.com/trebent/schemer"
	"github.com/xeipuuv/gojsonschema"
)

type (
	// Kerberos is the root configuration object for Kerberos.
	Kerberos struct {
		GatewayCfg     Gateway        `json:"gateway"`
		OASCfg         *OAS           `json:"oas,omitempty"`
		AuthCfg        *Auth          `json:"auth,omitempty"`
		AdminCfg       *Admin         `json:"admin,omitempty"`
		ObsCfg         *Observability `json:"observability,omitempty"`
		PersistenceCfg *Persistence   `json:"persistence,omitempty"`
	}

	// OAS holds configuration for OAS-based request routing and validation.
	OAS struct {
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

	// Gateway holds configuration for the API gateway.
	Gateway struct {
		Router *Router    `json:"router"`
		TLS    *ServerTLS `json:"tls,omitempty"`
	}

	// Router holds configuration for the request router.
	Router struct {
		Backends []*RouterBackend `json:"backends"`
	}

	// RouterBackend is a single router entry for a backend proxied by Kerberos.
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

	// Observability holds configuration for observability features.
	Observability struct {
		Enabled        bool `json:"enabled"`
		RuntimeMetrics bool `json:"runtimeMetrics"`
	}

	// Auth holds configuration for authentication and authorization.
	Auth struct {
		Methods *AuthMethods `json:"methods"`
		Scheme  *AuthScheme  `json:"scheme"`
		Order   int          `json:"order"`
	}

	// AuthMethods contain the configured methods of authentication and authorization.
	AuthMethods struct {
		Basic *AuthMethodBasic `json:"basic"`
	}

	// AuthMethodBasic is the Kerberos basic authentication method, where manages user sessions.
	AuthMethodBasic struct {
		API *AuthMethodBasicAPI `json:"api,omitempty"`
	}

	// AuthMethodBasicAPI contain settings for the administrative API side of the basic authentication mechanism.
	AuthMethodBasicAPI struct {
		Cookies *Cookies `json:"cookies,omitempty"`
		Origins *Origins `json:"origins,omitempty"`
	}

	// AuthScheme maps backends to AuthMethods.
	AuthScheme struct {
		Mappings []*AuthMapping `json:"mappings"`
	}

	// AuthMapping contain the mapping between backend and method, as well as custom overrides for authN and authZ.
	AuthMapping struct {
		Backend       string   `json:"backend"`
		Method        string   `json:"method"`
		Exempt        []string `json:"exempt"`
		Authorization *AuthZ   `json:"authorization"`
	}

	// AuthZ specifies the authorization scheme for a backend.
	AuthZ struct {
		Groups []string            `json:"groups"`
		Paths  map[string][]string `json:"paths"`
	}

	// Admin holds configuration for the admin API.
	Admin struct {
		SuperUser *SuperUser `json:"superUser"`
		API       *AdminAPI  `json:"api,omitempty"`
	}

	// SuperUser contains the provisioned credentials for the kerbeos super user.
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
)

//go:embed schemas/config_schema.json
var configSchema []byte

//go:embed schemas/admin_schema.json
var adminSchema []byte

//go:embed schemas/auth_schema.json
var authSchema []byte

//go:embed schemas/gateway_schema.json
var gatewaySchema []byte

//go:embed schemas/oas_schema.json
var oasSchema []byte

//go:embed schemas/observability_schema.json
var observabilitySchema []byte

//go:embed schemas/ordered_schema.json
var orderedSchema []byte

//go:embed schemas/router_schema.json
var routerSchema []byte

const defaultCalloutTimeoutMs = 5000

// NewKerberos returns a *Kerberos with default values set, note that you need to call PostProcess after parsing as well.
func NewKerberos() *Kerberos {
	return &Kerberos{
		AdminCfg:       newAdminConfig(),
		ObsCfg:         newObservabilityConfig(),
		PersistenceCfg: newPersistenceConfig(),
	}
}

// NewKerberosSchema returns a schemer.Schemer prepped with Kerberos schemas for validation and parsing.
func NewKerberosSchemer() *schemer.Schemer {
	return schemer.New(getKerberosSchema(), getKerberosSupportingSchemas()...)
}

func getKerberosSchema() gojsonschema.JSONLoader {
	return gojsonschema.NewBytesLoader(configSchema)
}

func getKerberosSupportingSchemas() []gojsonschema.JSONLoader {
	return []gojsonschema.JSONLoader{
		gojsonschema.NewBytesLoader(adminSchema),
		gojsonschema.NewBytesLoader(authSchema),
		gojsonschema.NewBytesLoader(gatewaySchema),
		gojsonschema.NewBytesLoader(oasSchema),
		gojsonschema.NewBytesLoader(observabilitySchema),
		gojsonschema.NewBytesLoader(orderedSchema),
		gojsonschema.NewBytesLoader(routerSchema),
		gojsonschema.NewBytesLoader(cookiesSchema),
		gojsonschema.NewBytesLoader(originsSchema),
		gojsonschema.NewBytesLoader(persistenceSchema),
	}
}

func newAdminConfig() *Admin {
	return &Admin{
		API: &AdminAPI{
			Cookies: &Cookies{
				SameSite: utilhttp.SameSiteStrict,
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

func newObservabilityConfig() *Observability {
	return &Observability{
		Enabled:        true,
		RuntimeMetrics: true,
	}
}

func newPersistenceConfig() *Persistence {
	return &Persistence{
		Driver:  "sqlite",
		Address: "krb.db",
	}
}

// PostProcess tweaks some default values after parsing.
func (kc *Kerberos) PostProcess() {
	kc.GatewayCfg.postProcess()

	if kc.OASCfg != nil {
		kc.OASCfg.postProcess()
	}

	if kc.AuthCfg != nil {
		kc.AuthCfg.postProcess()
	}

	kc.AdminCfg.postProcess()
	kc.ObsCfg.postProcess()
	kc.PersistenceCfg.postProcess()
}

// AuthEnabled returns true if auth is enabled.
func (kc *Kerberos) AuthEnabled() bool {
	return kc.AuthCfg != nil
}

// OASEnabled returns true if OAS validation is enabled.
func (kc *Kerberos) OASEnabled() bool {
	return kc.OASCfg != nil
}

func (ac *Auth) postProcess() {
	// Populate basic auth default values IF basic auth is enabled. This is done here since the whole "auth"
	// block is optional configuration.
	if ac.Methods.Basic != nil && ac.Methods.Basic.API == nil {
		ac.Methods.Basic.API = &AuthMethodBasicAPI{}
	}

	if ac.Methods.Basic != nil && ac.Methods.Basic.API.Cookies == nil {
		ac.Methods.Basic.API.Cookies = &Cookies{
			SameSite: utilhttp.SameSiteStrict,
		}
	}

	if ac.Methods.Basic != nil && ac.Methods.Basic.API.Origins == nil {
		ac.Methods.Basic.API.Origins = &Origins{}
	}
}

func (gc *Gateway) postProcess() {
	for _, b := range gc.Router.Backends {
		if b.TimeoutMs == 0 {
			b.TimeoutMs = defaultCalloutTimeoutMs
		}
	}
}
func (pc *Persistence) postProcess()   {}
func (oc *Observability) postProcess() {}
func (ac *Admin) postProcess()         {}
func (oc *OAS) postProcess() {
	for _, m := range oc.Mappings {
		if m.Options == nil {
			m.Options = &OASBackendMappingOpts{ValidateBody: true}
		}
	}
}
