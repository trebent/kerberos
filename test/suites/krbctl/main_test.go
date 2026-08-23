// Package krbctl contains the black-box integration suite for the krbctl CLI.
// Each test runs the real krbctl binary (located via the KRBCTL_BIN environment
// variable) in non-interactive mode and validates the generated files against
// the golden fixtures under testdata/.
package krbctl

import (
	"os"
	"testing"
)

// krbctlBin is the path to the krbctl binary under test, taken from KRBCTL_BIN.
var krbctlBin string

func TestMain(m *testing.M) {
	krbctlBin = os.Getenv("KRBCTL_BIN")
	if krbctlBin == "" {
		println("KRBCTL_BIN is not set; build krbctl and set KRBCTL_BIN to its path")
		os.Exit(1)
	}

	if _, err := os.Stat(krbctlBin); err != nil {
		println("KRBCTL_BIN does not point to an existing binary:", krbctlBin)
		os.Exit(1)
	}

	os.Exit(m.Run())
}
