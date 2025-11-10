package config

type (
	Map interface {
		Register(name string, cfg any, schemaPath string) error
		Resolve() error
		Access(name string) (any, error)
	}

	configEntry struct {
		schemaPath string
		cfg        any
	}

	impl struct {
		configEntries map[string]*configEntry
	}
)

const NoSchema = "no-schema"

func New() Map {
	return &impl{
		configEntries: make(map[string]*configEntry),
	}
}

func (c *impl) Register(name string, cfg any, schemaPath string) error {
	c.configEntries[name] = &configEntry{schemaPath, cfg}
	return nil
}

func (c *impl) Resolve() error {
	return nil
}

func (c *impl) Access(name string) (any, error) {
	return nil, nil
}
