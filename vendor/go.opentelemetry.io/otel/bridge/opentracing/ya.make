GO_LIBRARY()

LICENSE(Apache-2.0)

VERSION(v1.28.0)

SRCS(
    bridge.go
    doc.go
    provider.go
    util.go
    wrapper.go
)

GO_TEST_SRCS(
    bridge_test.go
    mix_test.go
    provider_test.go
)

END()

RECURSE(
    gotest
    internal
    migration
)
