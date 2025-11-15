package router

import (
	"github.com/kaptinlin/jsonschema"
	"github.com/trebent/kerberos/internal/config"
)

type routerCfg struct {
	Enabled bool `json:"enabled"`
}

func (o *routerCfg) Schema() *jsonschema.Schema {
	return config.NoSchema
}

const configName = "router"

var _ config.Config = &routerCfg{}

func RegisterWith(cfg config.Map) {
	cfg.Register(configName, &routerCfg{})
}
