GO_LIBRARY()

LICENSE(MIT)

VERSION(v0.0.0-20190221022429-1e957dd83bed)

SRCS(
    doc.go
    filehandler.go
    handler.go
    log.go
    logger.go
)

GO_TEST_SRCS(log_test.go)

END()

RECURSE(
    gotest
)
