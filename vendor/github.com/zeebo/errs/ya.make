GO_LIBRARY()

LICENSE(MIT)

VERSION(v1.4.0)

SRCS(
    errs.go
    group.go
    is_go1.20.go
)

GO_TEST_SRCS(
    errs_test.go
    group_test.go
)

END()

RECURSE(
    gotest
)
