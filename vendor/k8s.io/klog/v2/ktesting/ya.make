GO_LIBRARY()

LICENSE(Apache-2.0)

VERSION(v2.140.0)

SRCS(
    options.go
    setup.go
    testinglogger.go
)

END()

RECURSE(
    init
)
