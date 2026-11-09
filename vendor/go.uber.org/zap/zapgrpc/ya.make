GO_LIBRARY()

LICENSE(MIT)

VERSION(v1.28.0)

SRCS(
    zapgrpc.go
)

GO_TEST_SRCS(zapgrpc_test.go)

END()

RECURSE(
    gotest
)
