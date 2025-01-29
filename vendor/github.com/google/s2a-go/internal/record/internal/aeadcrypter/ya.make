GO_LIBRARY()

LICENSE(Apache-2.0)

VERSION(v0.1.7)

SRCS(
    aeadcrypter.go
    aesgcm.go
    chachapoly.go
    common.go
)

GO_TEST_SRCS(
    aesgcm_test.go
    chachapoly_test.go
    common_test.go
)

END()

RECURSE(
    gotest
    testutil
)
