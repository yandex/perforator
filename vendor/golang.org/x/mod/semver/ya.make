GO_LIBRARY()

LICENSE(BSD-3-Clause)

VERSION(v0.26.0)

SRCS(
    semver.go
)

GO_TEST_SRCS(semver_test.go)

END()

RECURSE(
    gotest
)
