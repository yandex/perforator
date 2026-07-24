GO_LIBRARY()

LICENSE(BSD-3-Clause)

VERSION(v0.270.0)

SRCS(
    internaloption.go
    unsaferesolver.go
)

GO_TEST_SRCS(
    internaloption_test.go
    unsaferesolver_test.go
)

GO_XTEST_SRCS(internaloption_external_test.go)

END()

RECURSE(
    gotest
)
