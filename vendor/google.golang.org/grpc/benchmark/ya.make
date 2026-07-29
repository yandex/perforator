GO_LIBRARY()

LICENSE(Apache-2.0)

VERSION(v1.80.0)

SRCS(
    benchmark.go
)

END()

RECURSE(
    benchmain
    benchresult
    client
    flags
    latency
    # primitives
    server
    stats
    worker
)
