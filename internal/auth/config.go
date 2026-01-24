package auth

import (
	_ "embed"

	"github.com/trebent/kerberos/internal/config"
	"github.com/xeipuuv/gojsonschema"
)

type (
	authConfig struct {
		Methods        *methods        `json:"methods"`
		Scheme         *scheme         `json:"scheme"`
		Administration *administration `json:"administration"`
		Order          int             `json:"order"`
	}
	methods struct {
		Basic *basicAuthentication `json:"basic"`
	}
	scheme struct {
		Mappings []*mapping `json:"mappings"`
	}
	mapping struct {
		Backend string `json:"backend"`
		Method  string `json:"method"`
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

const (
	configName = "auth"

	defaultSuperUserClientID     = "admin"
	defaultSuperUserClientSecret = "password"
	defaultOrder                 = 1
)

var (
	_ config.Config = &authConfig{}

	//go:embed config_schema.json
	schemaBytes []byte
)

func (a *authConfig) SchemaJSONLoader() gojsonschema.JSONLoader {
	return gojsonschema.NewBytesLoader(schemaBytes)
}

func (a *authConfig) BasicEnabled() bool {
	return a.Methods.Basic != nil
}

func (a *authConfig) AdministrationEnabled() bool {
	return a.Administration != nil
}

func RegisterWith(cfg config.Map) (string, error) {
	cfg.Register(configName, &authConfig{
		Administration: &administration{
			SuperUser: &superuser{
				ClientID:     defaultSuperUserClientID,
				ClientSecret: defaultSuperUserClientSecret,
			},
		},
		Order: defaultOrder,
	})
	return configName, nil
}
