GO_LIBRARY()

LICENSE(Apache-2.0)

VERSION(v0.20.2)

SRCS(
    cached.go
    flightcontrol.go
)

GO_TEST_SRCS(
    cached_test.go
    flightcontrol_test.go
)

END()

RECURSE(
    gotest
)
