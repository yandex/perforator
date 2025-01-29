GO_LIBRARY()

LICENSE(BSD-3-Clause)

VERSION(v2.19.1)

SRCS(
    grpc_api_configuration.go
    openapi_configuration.go
    registry.go
    services.go
    types.go
)

GO_TEST_SRCS(
    grpc_api_configuration_test.go
    openapi_configuration_test.go
    registry_test.go
    services_test.go
    types_test.go
)

END()

RECURSE(
    apiconfig
    gotest
    openapiconfig
)
