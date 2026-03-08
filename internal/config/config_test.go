package config

import (
	"os"
	"testing"

	"github.com/trebent/zerologr"
)

func TestConfigReferences(t *testing.T) {
	zerologr.Set(zerologr.New(&zerologr.Opts{V: 100, Console: true}))
	defer func() {
		zerologr.Set(zerologr.New(&zerologr.Opts{V: 0}))
	}()

	data, err := os.ReadFile("./testconfig/testconfig.json")
	if err != nil {
		t.Fatalf("failed to read test config: %v", err)
	}

	cfg := New()
	cfg.Load(data)
	if err := cfg.Parse(); err != nil {
		t.Fatalf("failed to load config: %v", err)
	}
}
