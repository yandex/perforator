GO_LIBRARY()

LICENSE(BSD-3-Clause)

VERSION(v0.53.0)

SRCS(
    client.go
    forward.go
    keyring.go
    server.go
)

GO_TEST_SRCS(
    client_pipeline_test.go
    client_test.go
    keyring_test.go
    server_test.go
    testdata_test.go
)

GO_XTEST_SRCS(example_test.go)

END()

RECURSE(
    gotest
)
