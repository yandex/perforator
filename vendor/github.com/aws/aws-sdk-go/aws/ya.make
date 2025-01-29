GO_LIBRARY()

LICENSE(Apache-2.0)

VERSION(v1.46.7)

SRCS(
    config.go
    context_1_9.go
    context_background_1_7.go
    context_sleep.go
    convert_types.go
    doc.go
    errors.go
    jsonvalue.go
    logger.go
    types.go
    url.go
    version.go
)

GO_TEST_SRCS(
    config_test.go
    convert_types_test.go
    types_test.go
)

GO_XTEST_SRCS(context_test.go)

END()

RECURSE(
    arn
    auth
    awserr
    awsutil
    client
    corehandlers
    credentials
    crr
    csm
    defaults
    ec2metadata
    endpoints
    # gotest
    request
    session
    signer
)
