GO_LIBRARY()

LICENSE(
    Apache-2.0 AND
    BSD-3-Clause
)

VERSION(v1.42.0)

SRCS(
    gen.go
    partialsuccess.go
    version.go
)

GO_TEST_SRCS(partialsuccess_test.go)

END()

RECURSE(
    counter
    envconfig
    gotest
    observ
    otlpconfig
    otlptracetest
    retry
    x
)
