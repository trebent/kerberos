package router

import (
	"github.com/trebent/kerberos/internal/config"
	"github.com/xeipuuv/gojsonschema"
)

type routerCfg struct {
	Enabled bool `json:"enabled"`
}

func (o *routerCfg) Schema() *gojsonschema.Schema {
	return config.NoSchema
}

const configName = "router"

var _ config.Config = &routerCfg{}

func RegisterWith(cfg config.Map) (string, error) {
	cfg.Register(configName, &routerCfg{})
	return configName, nil
}
