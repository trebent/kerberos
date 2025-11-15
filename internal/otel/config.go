package otel

import (
	"github.com/kaptinlin/jsonschema"
	"github.com/trebent/kerberos/internal/config"
)

type otelCfg struct {
	Enabled bool `json:"enabled"`
}

func (o *otelCfg) Schema() *jsonschema.Schema {
	return config.NoSchema
}

const configName = "otel"

var _ config.Config = &otelCfg{}

func RegisterWith(cfg config.Map) {
	cfg.Register(configName, &otelCfg{})
}
