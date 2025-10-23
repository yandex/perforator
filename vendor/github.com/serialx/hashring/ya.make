GO_LIBRARY()

LICENSE(MIT)

VERSION(v0.0.0-20200727003509-22c0c7ab6b1b)

SRCS(
    hash.go
    hashring.go
    key.go
)

GO_TEST_SRCS(
    allnodes_test.go
    benchmark_test.go
    example_test.go
    hashring_test.go
)

END()

RECURSE(
    gotest
)
