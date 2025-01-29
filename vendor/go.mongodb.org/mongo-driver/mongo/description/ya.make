GO_LIBRARY()

LICENSE(Apache-2.0)

VERSION(v1.17.1)

SRCS(
    description.go
    server.go
    server_kind.go
    server_selector.go
    topology.go
    topology_kind.go
    topology_version.go
    version_range.go
)

GO_TEST_SRCS(
    max_staleness_spec_test.go
    selector_spec_test.go
    selector_test.go
    server_test.go
    shared_spec_test.go
    version_range_test.go
)

END()

RECURSE(
    gotest
)
