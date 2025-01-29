GO_LIBRARY()

LICENSE(MIT)

VERSION(v0.11.1)

SRCS(
    mercator.go
)

GO_TEST_SRCS(mercator_test.go)

END()

RECURSE(
    gotest
)
