package router

import (
	_ "embed"

	"github.com/trebent/kerberos/internal/config"
	"github.com/xeipuuv/gojsonschema"
)

type routerConfig struct {
	Backends []*backend `json:"backends"`
}

const configName = "router"

var (
	_ config.Config = &routerConfig{}

	//go:embed config_schema.json
	schemaBytes []byte
)

func (o *routerConfig) Schema() *gojsonschema.Schema {
	s, err := gojsonschema.NewSchema(gojsonschema.NewBytesLoader(schemaBytes))
	if err != nil {
		panic("Failed to create schema for routerConfig: " + err.Error())
	}

	return s
}

func RegisterWith(cfg config.Map) (string, error) {
	cfg.Register(configName, &routerConfig{})
	return configName, nil
}
