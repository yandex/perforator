GO_LIBRARY()

LICENSE(Apache-2.0)

VERSION(v1.37.0)

SRCS(
    asyncfloat64.go
    asyncint64.go
    config.go
    doc.go
    instrument.go
    meter.go
    syncfloat64.go
    syncint64.go
)

GO_TEST_SRCS(
    asyncfloat64_test.go
    asyncint64_test.go
    instrument_test.go
    syncfloat64_test.go
    syncint64_test.go
)

GO_XTEST_SRCS(
    config_test.go
    # example_test.go
)

END()

RECURSE(
    embedded
    gotest
    noop
)
