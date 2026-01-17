package basicauth

//go:generate -command go tool -modfile=../../../tools/go.mod oapi-codegen -config ./config.yaml -o ./clientgen.go ../../../openapi/basic_auth.yaml
