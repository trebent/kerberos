package config

import (
	"os"
	"testing"

	"github.com/trebent/schemer"
)

func getConnectorSchemer(t *testing.T) *schemer.Schemer {
	t.Helper()
	s := schemer.New(getConnectorSchema(), getConnectorSupportingSchemas()...)
	return s
}

func TestConnector(t *testing.T) {
	t.Run("OK", func(t *testing.T) {
		sch := getConnectorSchemer(t)
		data, err := os.ReadFile("./testconfig/connector.json")
		checkErr(err, t)

		sch.Load(data)
		target := NewConnector()
		if err := sch.Parse(target); err != nil {
			t.Fatal(err.Error())
		}
	})

	t.Run("Missing persistence", func(t *testing.T) {
		sch := getConnectorSchemer(t)
		data, err := os.ReadFile("./testconfig/connector_no_persistence.json")
		checkErr(err, t)

		sch.Load(data)
		target := NewConnector()
		if err := sch.Parse(target); err == nil {
			t.Fatal("Should have errored out, missing persistence block")
		}
	})

	t.Run("Bad origins", func(t *testing.T) {
		sch := getConnectorSchemer(t)
		data, err := os.ReadFile("./testconfig/connector_bad_origins.json")
		checkErr(err, t)

		sch.Load(data)
		target := NewConnector()
		if err := sch.Parse(target); err == nil {
			t.Fatal("Should have errored out, bad origins block")
		}
	})
}
