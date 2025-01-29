GO_LIBRARY()

LICENSE(BSD-3-Clause)

VERSION(v0.176.1)

SRCS(
    internaloption.go
)

GO_TEST_SRCS(internaloption_test.go)

GO_XTEST_SRCS(internaloption_external_test.go)

END()

RECURSE(
    gotest
)
