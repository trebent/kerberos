package config

// ParseKerberos validates raw Kerberos configuration data against its schema, runs the
// semantic checks and applies post-processing defaults.
func ParseKerberos(data []byte) (*Kerberos, error) {
	cfg := NewKerberos()
	sch := NewKerberosSchemer()
	sch.Load(data)

	if err := sch.Parse(cfg); err != nil {
		return nil, err
	}

	if err := cfg.Validate(); err != nil {
		return nil, err
	}

	cfg.PostProcess()

	return cfg, nil
}

// ParseConnector validates raw admin-connector configuration data against its schema.
func ParseConnector(data []byte) (*Connector, error) {
	cfg := NewConnector()
	sch := NewConnectorSchemer()
	sch.Load(data)

	if err := sch.Parse(cfg); err != nil {
		return nil, err
	}

	return cfg, nil
}
