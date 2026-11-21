package config

import (
	"os"
	"testing"
)

func TestParseKerberos(t *testing.T) {
	if _, err := ParseKerberos(readTestConfig(t, "testconfig_gw.json")); err != nil {
		t.Fatal(err)
	}

	if _, err := ParseKerberos(readTestConfig(t, "testconfig_gw_router_duped_backend_name.json")); err == nil {
		t.Fatal("expected an error for a duplicate backend name")
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
