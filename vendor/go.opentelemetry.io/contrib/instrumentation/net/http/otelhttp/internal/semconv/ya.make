GO_LIBRARY()

LICENSE(Apache-2.0)

VERSION(v0.57.0)

SRCS(
    env.go
    httpconv.go
    util.go
    v1.20.0.go
)

GO_TEST_SRCS(
    bench_test.go
    common_test.go
    env_test.go
    httpconv_test.go
    util_test.go
    v1.20.0_test.go
)

END()

RECURSE(
    gotest
)
