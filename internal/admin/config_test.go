package admin

import (
	"testing"

	"github.com/trebent/kerberos/internal/config"
)

func TestConfigDefault(t *testing.T) {
	m := config.New(&config.Opts{})
	name, err := RegisterWith(m)
	if err != nil {
		t.Fatalf("Did not expect error: %v", err)
	}

	if name != configName {
		t.Fatalf("Expected config name %s, got %s", configName, name)
	}

	ac := config.AccessAs[*adminConfig](m, name)

	if ac.SuperUser.ClientID != "admin" {
		t.Fatal("Expected different default client ID")
	}

	if ac.SuperUser.ClientSecret != "secret" {
		t.Fatal("Expected different default client secret")
	}
}
