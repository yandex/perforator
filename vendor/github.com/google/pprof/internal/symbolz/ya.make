GO_LIBRARY()

LICENSE(Apache-2.0)

VERSION(v0.0.0-20240424215950-a892ee059fd6)

SRCS(
    symbolz.go
)

GO_TEST_SRCS(symbolz_test.go)

END()

RECURSE(
    gotest
)
