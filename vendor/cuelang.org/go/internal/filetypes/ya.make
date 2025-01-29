GO_LIBRARY()

LICENSE(Apache-2.0)

VERSION(v0.4.3)

SRCS(
    filetypes.go
    types.go
    util.go
)

GO_TEST_SRCS(
    filetypes_test.go
    util_test.go
)

END()

RECURSE(
    gotest
)
