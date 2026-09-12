package config

import (
	_ "embed"
)

type (
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
		// DenyAll denies any request with an Origin header, effectively disabling cross-site access.
		// Mutually exclusive with 'allowedOrigins' and 'allowAll'.
		DenyAll bool `json:"denyAll,omitempty"`
	}

	ServerTLS struct {
		CertFile string `json:"serverCertFile"`
		KeyFile  string `json:"serverKeyFile"`
	}

	// Persistence holds configuration for the backing database.
	Persistence struct {
		// Driver selects the database backend: "sqlite" or "postgres".
		Driver string `json:"driver"`
		// Address is the database address. For postgres: host. For sqlite: file path.
		Address string `json:"address"`

		// Postgres contains specific configuration for the postgres driver. Ignored for other drivers.
		*Postgres `json:"postgres,omitempty"`
	}

	// Postgres contains postgres-specific settings.
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

//go:embed schemas/cookies_schema.json
var cookiesSchema []byte

//go:embed schemas/origins_schema.json
var originsSchema []byte

//go:embed schemas/persistence_schema.json
var persistenceSchema []byte
