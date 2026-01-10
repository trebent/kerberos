package auth

import (
	_ "embed"

	"github.com/trebent/kerberos/internal/config"
	"github.com/xeipuuv/gojsonschema"
)

type (
	authConfig struct {
		Methods        *methods        `json:"methods"`
		Administration *administration `json:"administration"`
	}
	methods struct {
		Basic *basicAuthentication `json:"basic"`
	}
	administration struct {
		SuperUser *superuser `json:"superUser"`
	}
	superuser struct {
		ClientID     string `json:"clientId"`
		ClientSecret string `json:"clientSecret"`
	}
	basicAuthentication struct{}
)

const configName = "auth"

var (
	_ config.Config = &authConfig{}

	//go:embed config_schema.json
	schemaBytes []byte
)

func (a *authConfig) Schema() *gojsonschema.Schema {
	s, err := gojsonschema.NewSchema(gojsonschema.NewBytesLoader(schemaBytes))
	if err != nil {
		panic("Failed to create schema for authConfig: " + err.Error())
	}

	return s
}

func (a *authConfig) BasicEnabled() bool {
	return a.Methods.Basic != nil
}

func (a *authConfig) AdministrationEnabled() bool {
	return a.Administration != nil
}

func RegisterWith(cfg config.Map) (string, error) {
	cfg.Register(configName, &authConfig{})
	return configName, nil
}
