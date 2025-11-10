package config

import "testing"

type testCfg struct {
	Enabled bool `json:"enabled"`
}

func (t *testCfg) Schema() string {
	return NoSchema
}

func TestLoadNoName(t *testing.T) {
	m := New()

	m.Register("ok", &testCfg{})

	if err := m.Load("ok", []byte{}); err != nil {
	}

	if err := m.Load("nok", []byte{}); err == nil {
		t.Fatal("Should have errored out due to no registered config name")
	}
}
