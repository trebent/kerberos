package config

import (
	"errors"
	"fmt"
)

type (
	Map interface {
		Register(name string, cfg Config)
		Load(name string, data []byte) error
		Validate() error
		Access(name string) (any, error)
	}
	Config interface {
		Schema() string
	}

	configEntry struct {
		schemaPath string
		cfg        Config
		data       []byte
	}

	impl struct {
		configEntries map[string]*configEntry
	}
)

const NoSchema = "no-schema"

var ErrNoRegisteredName = errors.New("could not find a config entry with that name")

func New() Map {
	return &impl{
		configEntries: make(map[string]*configEntry),
	}
}

func (c *impl) Register(name string, cfg Config) {
	c.configEntries[name] = &configEntry{cfg.Schema(), cfg, nil}
}

func (c *impl) Load(name string, data []byte) error {
	entry, ok := c.configEntries[name]
	if !ok {
		return fmt.Errorf("%w: %s", ErrNoRegisteredName, name)
	}

	entry.data = data

	return nil
}

func (c *impl) Validate() error {
	return nil
}

func (c *impl) Access(name string) (any, error) {
	_ = c.configEntries[name]

	return nil, nil
}
