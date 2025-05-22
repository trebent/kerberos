package router

import (
	"encoding/json"
	"os"

	"github.com/trebent/zerologr"
)

type (
	BackendLoader interface {
		Load() ([]Backend, error)
	}
	jsonLoader struct {
		file string
	}
	jsonFile struct {
		Backends []*backend `json:"backends"`
	}
)

var _ BackendLoader = &jsonLoader{}

func NewJSONLoader(filePath string) BackendLoader {
	return &jsonLoader{file: filePath}
}

func (j *jsonLoader) Load() ([]Backend, error) {
	data, err := os.ReadFile(j.file)
	if err != nil {
		return nil, err
	}

	zerologr.Info(string(data))

	fileConfig := &jsonFile{}
	if err := json.Unmarshal(data, &fileConfig); err != nil {
		return nil, err
	}

	backends := make([]Backend, len(fileConfig.Backends))
	for i, backend := range fileConfig.Backends {
		backends[i] = backend
	}

	return backends, nil
}
