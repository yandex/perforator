GO_LIBRARY()

LICENSE(MIT)

VERSION(v1.3.0)

SRCS(
    doc.go
    leaks.go
    options.go
    testmain.go
    tracestack_new.go
)

GO_TEST_SRCS(
    leaks_test.go
    options_test.go
    testmain_test.go
    utils_test.go
)

GO_XTEST_SRCS(signal_test.go)

END()

RECURSE(
    gotest
    internal
)
