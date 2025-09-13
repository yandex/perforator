GO_LIBRARY()

LICENSE(BSD-3-Clause)

VERSION(v0.228.0)

SRCS(
    googleapi.go
    types.go
)

GO_TEST_SRCS(
    googleapi_test.go
    types_test.go
)

END()

RECURSE(
    gotest
    transport
)
