GO_LIBRARY()

LICENSE(
    Apache-2.0 AND
    BSD-3-Clause
)

VERSION(v1.38.0)

SRCS(
    clients.go
    doc.go
    exporter.go
    version.go
)

GO_XTEST_SRCS(
    exporter_test.go
    version_test.go
)

END()

RECURSE(
    gotest
    internal
)
