GO_LIBRARY()

LICENSE(BSD-2-Clause)

VERSION(v1.14.0)

SRCS(
    json_pointer.go
    string.go
    tag.go
    utils.go
)

GO_XTEST_SRCS(
    json_pointer_test.go
    string_test.go
    tag_test.go
    utils_test.go
)

END()

RECURSE(
    gotest
)
