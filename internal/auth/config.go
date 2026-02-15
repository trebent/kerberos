package auth

import (
	_ "embed"

	"github.com/trebent/kerberos/internal/auth/method"
	"github.com/trebent/kerberos/internal/config"
	"github.com/xeipuuv/gojsonschema"
)

type (
	authConfig struct {
		Methods *methods `json:"methods"`
		Scheme  *scheme  `json:"scheme"`
		Order   int      `json:"order"`
	}
	methods struct {
		Basic *basicAuthentication `json:"basic"`
	}
	scheme struct {
		Mappings []*mapping `json:"mappings"`
	}
	mapping struct {
		Backend       string             `json:"backend"`
		Method        string             `json:"method"`
		Exempt        []string           `json:"exempt"`
		Authorization method.AuthZConfig `json:"authorization"`
	}
	basicAuthentication struct{}
)

const (
	configName = "auth"

	defaultOrder = 1
)

var (
	_ config.Config = &authConfig{}

	//go:embed config_schema.json
	schemaBytes []byte
)

// SchemaJSONLoader implements [config.Config].
func (a *authConfig) SchemaJSONLoader() gojsonschema.JSONLoader {
	return gojsonschema.NewBytesLoader(schemaBytes)
}

func (a *authConfig) BasicEnabled() bool {
	return a.Methods.Basic != nil
}

func RegisterWith(cfg config.Map) (string, error) {
	cfg.Register(configName, &authConfig{Order: defaultOrder})
	return configName, nil
}
