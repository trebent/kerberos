package otel

import (
	"github.com/trebent/kerberos/internal/config"
	"github.com/xeipuuv/gojsonschema"
)

type otelCfg struct {
	Enabled bool `json:"enabled"`
}

func (o *otelCfg) Schema() *gojsonschema.Schema {
	return config.NoSchema
}

const configName = "otel"

var _ config.Config = &otelCfg{}

func RegisterWith(cfg config.Map) (string, error) {
	cfg.Register(configName, &otelCfg{})
	return configName, nil
}
