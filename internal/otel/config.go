package otel

import "github.com/trebent/kerberos/internal/config"

type otelCfg struct {
	Enabled bool `json:"enabled"`
}

const configName = "otel"

func RegisterWith(cfg config.Map) error {
	return cfg.Register(configName, &otelCfg{}, config.NoSchema)
}
