GO_LIBRARY()

SRCS(
    conf.go
    map.go
    parser.go
    registry.go
)

GO_TEST_SRCS(
    conf_test.go
    parser_test.go
)

END()

RECURSE(
    gotest
)
