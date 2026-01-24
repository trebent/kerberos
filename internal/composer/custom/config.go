package custom

import (
	_ "embed"

	"github.com/xeipuuv/gojsonschema"
)

//go:embed ordered_schema.json
var orderedSchema []byte

func OrderedSchemaJSONLoader() gojsonschema.JSONLoader {
	return gojsonschema.NewBytesLoader(orderedSchema)
}
