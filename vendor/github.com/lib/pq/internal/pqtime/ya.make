GO_LIBRARY()

LICENSE(MIT)

VERSION(v1.12.0)

SRCS(
    loc.go
    pqtime.go
)

GO_TEST_SRCS(loc_test.go)

GO_XTEST_SRCS(pqtime_test.go)

END()

RECURSE(
    gotest
)
