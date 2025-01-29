GO_LIBRARY()

SRCS(
    downloader.go
)

GO_TEST_SRCS(downloader_test.go)

END()

RECURSE(
    gotest
)
