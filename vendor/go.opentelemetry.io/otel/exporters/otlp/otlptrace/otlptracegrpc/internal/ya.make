GO_LIBRARY()

LICENSE(
    Apache-2.0 AND
    BSD-3-Clause
)

VERSION(v1.38.0)

SRCS(
    gen.go
    partialsuccess.go
)

GO_TEST_SRCS(partialsuccess_test.go)

END()

RECURSE(
    envconfig
    gotest
    otlpconfig
    otlptracetest
    retry
)
