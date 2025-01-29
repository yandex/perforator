GO_LIBRARY()

LICENSE(BSD-3-Clause)

VERSION(v0.10.0)

# skipped due to run_go_test bug: Test output parser error: Trying to run the test [TestPanicDoChan] which has been already started...

GO_SKIP_TESTS(
    TestPanicDoChan
    TestPanicDoSharedByDoChan
)

SRCS(
    singleflight.go
)

GO_TEST_SRCS(singleflight_test.go)

END()

RECURSE(
    gotest
)
