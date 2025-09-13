GO_LIBRARY()

LICENSE(MIT)

VERSION(v1.4.0)

SRCS(
    reflect.go
)

GO_TEST_SRCS(reflect_test.go)

END()

RECURSE(
    gotest
)
