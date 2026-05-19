GO_LIBRARY()

LICENSE(MIT)

VERSION(v0.12.0)

SRCS(
    mercator.go
)

GO_TEST_SRCS(mercator_test.go)

END()

RECURSE(
    gotest
)
