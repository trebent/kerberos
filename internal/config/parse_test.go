package config

import (
	"os"
	"strings"
	"testing"
)

func TestParseKerberos(t *testing.T) {
	for _, name := range []string{"testconfig_gw.json", "testconfig.json", "testconfig_oas.json"} {
		t.Run(name, func(t *testing.T) {
			if _, err := ParseKerberos(readTestConfig(t, name)); err != nil {
				t.Fatal(err)
			}
		})
	}

	if _, err := ParseKerberos(readTestConfig(t, "testconfig_gw_router_duped_backend_name.json")); err == nil {
		t.Fatal("expected an error for a duplicate backend name")
	}
}

func TestParseKerberosUnknownMappedBackend(t *testing.T) {
	tests := []struct {
		file string
		want string
	}{
		{file: "testconfig_auth_unknown_backend.json", want: `auth.scheme.mappings.1.backend - unknown backend "bakcend1"`},
		{file: "testconfig_oas_unknown_backend.json", want: `oas.mappings.0.backend - unknown backend "bakcend1"`},
	}

	for _, tt := range tests {
		t.Run(tt.file, func(t *testing.T) {
			_, err := ParseKerberos(readTestConfig(t, tt.file))
			if err == nil {
				t.Fatal("expected an error for a mapping that names an unknown backend")
			}

			if !strings.Contains(err.Error(), tt.want) {
				t.Fatalf("expected error to contain %q, got %q", tt.want, err.Error())
			}
		})
	}
}

func TestParseConnector(t *testing.T) {
	if _, err := ParseConnector(readTestConfig(t, "connector.json")); err != nil {
		t.Fatal(err)
	}

	if _, err := ParseConnector(readTestConfig(t, "connector_no_persistence.json")); err == nil {
		t.Fatal("expected an error for a missing persistence block")
	}
}

func readTestConfig(t *testing.T, name string) []byte {
	t.Helper()

	data, err := os.ReadFile("./testconfig/" + name)
	checkErr(err, t)

	return data
}
