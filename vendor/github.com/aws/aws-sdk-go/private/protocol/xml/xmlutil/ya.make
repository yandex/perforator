GO_LIBRARY()

LICENSE(Apache-2.0)

VERSION(v1.46.7)

SRCS(
    build.go
    sort.go
    unmarshal.go
    xml_to_struct.go
)

GO_TEST_SRCS(
    build_test.go
    sort_test.go
    unmarshal_test.go
)

END()

RECURSE(
    gotest
)
