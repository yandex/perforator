GO_LIBRARY()

SRCS(
    cpulist.go
)

GO_TEST_SRCS(cpulist_test.go)

END()

RECURSE(
    gotest
)
