GO_LIBRARY()

SRCS(
    clock.go
    lastday.go
    parse.go
    special.go
    unix.go
    wellknown.go
)

GO_TEST_SRCS(parse_test.go)

END()

RECURSE(
    gotest
)
