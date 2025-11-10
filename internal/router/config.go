package router

import "github.com/trebent/kerberos/internal/config"

type routerCfg struct {
	Enabled bool `json:"enabled"`
}

const configName = "router"

func RegisterWith(cfg config.Map) error {
	return cfg.Register(configName, &routerCfg{}, config.NoSchema)
}
