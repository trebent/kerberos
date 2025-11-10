package config

type (
	Map interface {
		Register(name string, cfg any) error
		Resolve() error
		Access(name string) (any, error)
	}
	impl struct {
	}
)

func New() Map {
	return &impl{}
}

func (c *impl) Register(name string, cfg any) error {
	return nil
}

func (c *impl) Resolve() error {
	return nil
}

func (c *impl) Access(name string) (any, error) {
	return nil, nil
}
