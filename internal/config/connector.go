package config

import (
	_ "embed"

	"github.com/trebent/schemer"
	"github.com/xeipuuv/gojsonschema"
)

type (
	Connector struct {
		// Origins holds configuration for CORS origins.
		Origins *Origins `json:"origins,omitempty"`

		// Persistence holds configuration for the persistence layer.
		Persistence *Persistence `json:"persistence"`

		// TLS holds configuration for the server TLS settings.
		TLS *ServerTLS `json:"tls,omitempty"`

		// TargetTLS holds configuration for the target TLS settings.
		TargetTLS *TargetTLS `json:"targetTls,omitempty"`
	}

	TargetTLS struct {
		// RootCAFile is the path to a PEM-encoded CA bundle used to verify the target's certificate.
		// When empty, the system certificate pool is used.
		RootCAFile string `json:"rootCAFile,omitempty"`
		// InsecureSkipVerify indicates whether to skip TLS verification for the target.
		InsecureSkipVerify bool `json:"insecureSkipVerify,omitempty"`
	}
)

//go:embed schemas/connector_schema.json
var connectorSchema []byte

// NewConnector returns a *Connector config struct with defaults set.
func NewConnector() *Connector {
	return &Connector{
		Origins: &Origins{},
	}
}

// NewConnectorSchemer returns a schemer.Schemer prepped with Connector schemas for validation and parsing.
func NewConnectorSchemer() *schemer.Schemer {
	return schemer.New(getConnectorSchema(), getConnectorSupportingSchemas()...)
}

func getConnectorSchema() gojsonschema.JSONLoader {
	return gojsonschema.NewBytesLoader(connectorSchema)
}

func getConnectorSupportingSchemas() []gojsonschema.JSONLoader {
	return []gojsonschema.JSONLoader{
		gojsonschema.NewBytesLoader(originsSchema),
		gojsonschema.NewBytesLoader(persistenceSchema),
	}
}
