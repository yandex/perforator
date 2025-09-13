GO_LIBRARY()

LICENSE(BSD-3-Clause)

VERSION(v0.15.0)

SRCS(
    semaphore.go
)

GO_XTEST_SRCS(
    semaphore_bench_test.go
    semaphore_example_test.go
    semaphore_test.go
)

END()

RECURSE(
    gotest
)
