GO_LIBRARY()

LICENSE(Apache-2.0)

VERSION(v0.5.2)

SRCS(
    doc.go
    mock.go
    vendor_dep.go
)

END()

RECURSE(
    source_mock_package
)
