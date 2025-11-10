package otel

import "github.com/trebent/kerberos/internal/config"

type otelCfg struct {
	Enabled bool `json:"enabled"`
}

func (o *otelCfg) Schema() string {
	return config.NoSchema
}

const configName = "otel"

var _ config.Config = &otelCfg{}

func RegisterWith(cfg config.Map) {
	cfg.Register(configName, &otelCfg{})
}
