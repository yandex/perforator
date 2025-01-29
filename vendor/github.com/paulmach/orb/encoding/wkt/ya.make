GO_LIBRARY()

LICENSE(MIT)

VERSION(v0.11.1)

SRCS(
    unmarshal.go
    wkt.go
)

GO_TEST_SRCS(
    benchmarks_test.go
    unmarshal_test.go
    wkt_test.go
)

END()

RECURSE(
    gotest
)
