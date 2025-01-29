GO_LIBRARY()

LICENSE(MIT)

VERSION(v1.3.0)

SRCS(
    doc.go
    scan.go
    stacks.go
)

GO_TEST_SRCS(
    scan_test.go
    stacks_test.go
)

END()

RECURSE(
    gotest
)
