package router

import "github.com/trebent/kerberos/internal/config"

type routerCfg struct {
	Enabled bool `json:"enabled"`
}

func (o *routerCfg) Schema() string {
	return config.NoSchema
}

const configName = "router"

var _ config.Config = &routerCfg{}

func RegisterWith(cfg config.Map) {
	cfg.Register(configName, &routerCfg{})
}
