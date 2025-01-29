GO_LIBRARY()

SRCS(
    pprof.go
)

GO_XTEST_SRCS(pprof_test.go)

END()

RECURSE(
    gotest
)
