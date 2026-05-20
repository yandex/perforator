GO_LIBRARY()

LICENSE(MIT)

VERSION(v0.32.0)

SRCS(
    network.go
)

GO_XTEST_SRCS(
    # network_test.go
)

END()

RECURSE(
    gotest
)
