GO_LIBRARY()

LICENSE(Apache-2.0)

VERSION(v0.46.0)

SRCS(
    autoneg.go
)

GO_TEST_SRCS(autoneg_test.go)

END()

RECURSE(
    gotest
)
