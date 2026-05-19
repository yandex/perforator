GO_LIBRARY()

LICENSE(MIT)

VERSION(v0.9.0)

SRCS(
    envelope.go
    sign.go
    signerverifier.go
    verify.go
)

GO_TEST_SRCS(
    sign_test.go
    verify_test.go
)

END()

RECURSE(
    gotest
)
