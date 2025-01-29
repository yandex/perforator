GO_LIBRARY()

LICENSE(MIT)

VERSION(v1.3.5)

SRCS(
    reflect.go
)

GO_TEST_SRCS(reflect_test.go)

END()

RECURSE(
    gotest
)
