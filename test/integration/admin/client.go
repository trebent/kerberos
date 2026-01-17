package admin

//go:generate -command go tool -modfile=../../../tools/go.mod oapi-codegen -config ./config.yaml -o ./clientgen.go ../../../openapi/administration.yaml
