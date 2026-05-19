GO_LIBRARY()

LICENSE(Apache-2.0)

VERSION(v0.13.2)

SRCS(
    flightcontrol.go
)

GO_TEST_SRCS(flightcontrol_test.go)

END()

RECURSE(
    gotest
)
