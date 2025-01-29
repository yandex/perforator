GO_LIBRARY()

LICENSE(MIT)

VERSION(v1.1.0)

SRCS(
    doc.go
    env.go
    gostub.go
    version.go
)

GO_TEST_SRCS(
    assignable_test.go
    env_test.go
    func_test.go
    gostub_test.go
    utils_for_test.go
)

GO_XTEST_SRCS(
    examples_test.go
    examples_time_const_test.go
    examples_time_func_test.go
)

END()

RECURSE(
    gotest
)
