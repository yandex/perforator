GO_LIBRARY()

LICENSE(Apache-2.0)

VERSION(v1.17.1)

SRCS(
    connstring.go
)

GO_XTEST_SRCS(
    connstring_spec_test.go
    connstring_test.go
)

END()

RECURSE(
    gotest
)
