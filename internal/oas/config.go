package oas

import (
	_ "embed"

	"github.com/trebent/kerberos/internal/config"
	"github.com/xeipuuv/gojsonschema"
)

type (
	oasConfig struct {
		Mappings []*mapping `json:"mappings"`
		Order    int        `json:"order"`
	}
	mapping struct {
		Backend       string `json:"backend"`
		Specification string `json:"specification"`
	}
)

const (
	configName = "oas"
)

var (
	_ config.Config = &oasConfig{}

	//go:embed config_schema.json
	schemaBytes []byte
)

// SchemaJSONLoader implements [config.Config].
func (o *oasConfig) SchemaJSONLoader() gojsonschema.JSONLoader {
	return gojsonschema.NewBytesLoader(schemaBytes)
}

func RegisterWith(cm config.Map) (string, error) {
	cm.Register(configName, &oasConfig{
		Order: 1,
	})
	return configName, nil
}
