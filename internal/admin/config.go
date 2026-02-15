package admin

import (
	_ "embed"
	"errors"

	"github.com/trebent/kerberos/internal/config"
	"github.com/xeipuuv/gojsonschema"
)

type (
	adminConfig struct {
		SuperUser superUser `json:"superUser"`
	}
	superUser struct {
		ClientID     string `json:"clientId"`
		ClientSecret string `json:"clientSecret"`
	}
)

const configName = "admin"

var (
	_ config.Config = &adminConfig{}

	//go:embed config_schema.json
	schemaBytes []byte

	ErrNoConfig = errors.New(
		"kerberos administration API is using default settings, this is NOT safe for production",
	)
)

// SchemaJSONLoader implements [config.Config].
func (a *adminConfig) SchemaJSONLoader() gojsonschema.JSONLoader {
	return gojsonschema.NewBytesLoader(schemaBytes)
}

func RegisterWith(cfg config.Map) (string, error) {
	cfg.Register(configName, &adminConfig{SuperUser: superUser{
		ClientID: "admin", ClientSecret: "secret",
	}})
	return configName, nil
}
