GO_LIBRARY()

LICENSE(
    Apache-2.0 AND
    BSD-3-Clause
)

VERSION(v1.40.0)

SRCS(
    features.go
    x.go
)

GO_TEST_SRCS(
    features_test.go
    x_test.go
)

END()

RECURSE(
    gotest
)
