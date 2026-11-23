GO_LIBRARY()

LICENSE(MIT)

VERSION(v1.9.4)

SRCS(
    test.go
)

GO_TEST_SRCS(test_test.go)

END()

RECURSE(
    gotest
)
