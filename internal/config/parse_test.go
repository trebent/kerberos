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
		{file: "testconfig_auth_method_not_configured.json", want: `auth.scheme.mappings.0.method - method "basic" is not configured`},
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

func TestParseKerberosIncompleteSections(t *testing.T) {
	for _, name := range []string{"testconfig_gw_no_router.json", "testconfig_auth_empty.json"} {
		t.Run(name, func(t *testing.T) {
			if _, err := ParseKerberos(readTestConfig(t, name)); err == nil {
				t.Fatal("expected an error for an incomplete section")
			}
		})
	}
}

func TestParseKerberosSuperUserSecret(t *testing.T) {
	const gateway = `"gateway": {"router": {"backends": [{"name": "a", "host": "h", "port": 1}]}}`

	tests := map[string]struct {
		admin   string
		wantErr bool
	}{
		"no admin section": {admin: "", wantErr: true},
		"no superuser":     {admin: `, "admin": {}`, wantErr: true},
		"no secret":        {admin: `, "admin": {"superUser": {}}`, wantErr: true},
		"secret too short": {admin: `, "admin": {"superUser": {"clientSecret": "123456789"}}`, wantErr: true},
		"secret too long":  {admin: `, "admin": {"superUser": {"clientSecret": "` + strings.Repeat("a", 65) + `"}}`, wantErr: true},
		"shortest secret":  {admin: `, "admin": {"superUser": {"clientSecret": "1234567890"}}`},
		"longest secret":   {admin: `, "admin": {"superUser": {"clientSecret": "` + strings.Repeat("a", 64) + `"}}`},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			_, err := ParseKerberos([]byte("{" + gateway + tt.admin + "}"))
			if tt.wantErr && err == nil {
				t.Fatal("expected an error")
			}

			if !tt.wantErr && err != nil {
				t.Fatalf("expected no error, got %v", err)
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
