package obs

import (
	_ "embed"

	"github.com/trebent/kerberos/internal/config"
	"github.com/xeipuuv/gojsonschema"
)

type obsConfig struct {
	Enabled        bool `json:"enabled"`
	RuntimeMetrics bool `json:"runtimeMetrics"`
}

const configName = "observability"

var (
	_ config.Config = &obsConfig{}

	//go:embed config_schema.json
	schemaBytes []byte
)

func (o *obsConfig) SchemaJSONLoader() gojsonschema.JSONLoader {
	return gojsonschema.NewBytesLoader(schemaBytes)
}

func RegisterWith(cfg config.Map) (string, error) {
	cfg.Register(configName, &obsConfig{
		Enabled:        true,
		RuntimeMetrics: true,
	})
	return configName, nil
}
