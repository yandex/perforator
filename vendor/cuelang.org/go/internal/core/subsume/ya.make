GO_LIBRARY()

LICENSE(Apache-2.0)

VERSION(v0.4.3)

SRCS(
    structural.go
    subsume.go
    value.go
    vertex.go
)

GO_TEST_SRCS(
    structural_test.go
    subsume_test.go
    value_test.go
)

END()

RECURSE(
    gotest
)
