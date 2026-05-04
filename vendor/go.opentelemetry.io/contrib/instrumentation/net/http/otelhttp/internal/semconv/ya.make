GO_LIBRARY()

LICENSE(
    Apache-2.0 AND
    BSD-3-Clause
)

VERSION(v0.63.0)

SRCS(
    env.go
    gen.go
    httpconv.go
    util.go
)

GO_TEST_SRCS(
    bench_test.go
    env_test.go
    httpconv_test.go
    util_test.go
)

GO_XTEST_SRCS(
    common_test.go
    httpconvtest_test.go
)

END()

RECURSE(
    gotest
)
