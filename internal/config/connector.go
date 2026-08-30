package config

type ConnectorConfig struct {
	// Origins holds configuration for CORS origins.
	Origins *Origins `json:"origins,omitempty"`

	// Persistence holds configuration for the persistence layer.
	Persistence *PersistenceConfig `json:"persistence"`

	// TLS holds configuration for the server TLS settings.
	TLS *ServerTLS `json:"tls,omitempty"`

	// TargetTLS holds configuration for the target TLS settings.
	TargetTLS *TargetTLS `json:"targetTls,omitempty"`
}

type TargetTLS struct {
	// RootCAFile is the path to a PEM-encoded CA bundle used to verify the target's certificate.
	// When empty, the system certificate pool is used.
	RootCAFile string `json:"rootCAFile,omitempty"`
	// InsecureSkipVerify indicates whether to skip TLS verification for the target.
	InsecureSkipVerify bool `json:"insecureSkipVerify,omitempty"`
}
