GO_LIBRARY()

SRCS(
    fs.go
)

GO_TEST_SRCS(
    fs_test.go
)

END()

RECURSE(
    gotest
)
