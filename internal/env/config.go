//nolint:gochecknoglobals // Package env provides environment variable parsing for the Kerberos service.
package env

import (
	"os"

	"github.com/trebent/envparser"
)

var (
	RouteJSONFile = envparser.Register(&envparser.Opts[string]{
		Name:  "ROUTE_JSON_FILE",
		Desc:  "JSON file to load routes from.",
		Value: "./routes.json",
		Validate: func(path string) error {
			f, err := os.Open(path)
			if err != nil {
				return err
			}
			defer f.Close()
			return nil
		},
	})
	ObsJSONFile = envparser.Register(&envparser.Opts[string]{
		Name:  "OBS_JSON_FILE",
		Desc:  "JSON file to load observability settings from.",
		Value: "",
		Validate: func(path string) error {
			if len(path) == 0 {
				return nil
			}

			f, err := os.Open(path)
			if err != nil {
				return err
			}
			defer f.Close()
			return nil
		},
	})
)
