GO_LIBRARY()

LICENSE(
    Apache-2.0 AND
    BSD-3-Clause
)

VERSION(v1.42.0)

SRCS(
    observ.go
    x.go
)

GO_TEST_SRCS(
    observ_test.go
    x_test.go
)

END()

RECURSE(
    gotest
)
