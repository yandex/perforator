GO_LIBRARY()

LICENSE(Apache-2.0)

VERSION(v1.63.2)

SRCS(
    google.go
    xds.go
)

GO_TEST_SRCS(
    google_test.go
    xds_test.go
)

END()

RECURSE(
    gotest
)
