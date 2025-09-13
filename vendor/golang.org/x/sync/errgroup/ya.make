GO_LIBRARY()

LICENSE(BSD-3-Clause)

VERSION(v0.15.0)

SRCS(
    errgroup.go
)

GO_XTEST_SRCS(
    errgroup_example_md5all_test.go
    errgroup_test.go
)

END()

RECURSE(
    gotest
)
