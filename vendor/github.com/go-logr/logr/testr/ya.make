GO_LIBRARY()

LICENSE(Apache-2.0)

VERSION(v1.4.3)

SRCS(
    testr.go
)

GO_TEST_SRCS(
    testr_fuzz_test.go
    testr_test.go
)

END()

RECURSE(
    gotest
)
