package main

import (
	"testing"

	"github.com/trebent/envparser"
	"github.com/trebent/kerberos/internal/docscheck"
)

// TestEnvDocs keeps the environment variable table in docs/deployment.md, the one place these
// variables are documented, in step with env.go.
func TestEnvDocs(t *testing.T) {
	docscheck.CheckEnvTable(t, "docs/deployment.md", envparser.Help())
}
