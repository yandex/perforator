GO_LIBRARY()

LICENSE(Apache-2.0)

VERSION(v1.0.0)

SRCS(
    algorithm.go
    digest.go
    digester.go
    doc.go
    verifiers.go
)

GO_TEST_SRCS(
    algorithm_test.go
    digest_test.go
    verifiers_test.go
)

END()

RECURSE(
    digestset
    gotest
)
