package config

type ConnectorConfig struct {
	// Whitelist is a list of allowed hosts for the connector.
	Whitelist []string `json:"whitelist,omitempty"`

	// Persistence holds configuration for the persistence layer.
	Persistence *PersistenceConfig `json:"persistence"`

	// ServerTLS holds configuration for the server TLS settings.
	ServerTLS *ServerTLS `json:"serverTls,omitempty"`
}
