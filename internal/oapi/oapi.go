// Package oapi contains the OpenAPI generated API types for the Kerberos project. Generated
// code is found in subpackages, each corresponding to the different OpenAPI specifications
// defined in the project. Each subpackage contains the types and optionally client code
// generated from its respective OpenAPI spec.
// It also contains general error handling types used across all APIs.
package oapi

//go:generate oapi-codegen -config auth_basic_config.yaml ../../openapi/auth_basic.yaml
//go:generate oapi-codegen -config admin_config.yaml ../../openapi/admin.yaml
//go:generate oapi-codegen -config gateway_config.yaml ../../openapi/gateway.yaml
