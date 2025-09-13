GO_LIBRARY()

LICENSE(Apache-2.0)

VERSION(v1.73.0)

SRCS(
    config.go
    logging.go
    picker.go
    ring.go
    ringhash.go
)

GO_TEST_SRCS(
    config_test.go
    picker_test.go
    ring_test.go
    ringhash_test.go
)

GO_XTEST_SRCS(
    # ringhash_e2e_test.go
)

END()

RECURSE(
    gotest
)
