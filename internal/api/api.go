// Package api contains the OpenAPI generated API types for the Kerberos project.
// It also contains general error handling types used across all APIs.
package api

//go:generate oapi-codegen -config auth_basic_config.yaml ../../openapi/auth_basic.yaml
//go:generate oapi-codegen -config admin_config.yaml ../../openapi/admin.yaml
//go:generate oapi-codegen -config gateway_config.yaml ../../openapi/gateway.yaml
