GO_LIBRARY()

LICENSE(BSD-3-Clause)

VERSION(v0.26.0)

SRCS(
    print.go
    read.go
    rule.go
    work.go
)

GO_TEST_SRCS(
    read_test.go
    rule_test.go
    work_test.go
)

END()

RECURSE(
    gotest
)
