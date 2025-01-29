GO_LIBRARY()

LICENSE(Apache-2.0)

VERSION(v1.17.1)

SRCS(
    batch_cursor.go
    batches.go
    compression.go
    crypt.go
    driver.go
    errors.go
    legacy.go
    operation.go
    operation_exhaust.go
    serverapioptions.go
)

GO_TEST_SRCS(
    batch_cursor_test.go
    batches_test.go
    command_monitoring_test.go
    compression_test.go
    operation_test.go
)

END()

RECURSE(
    auth
    connstring
    dns
    drivertest
    gotest
    #integration
    mongocrypt
    ocsp
    operation
    session
    topology
    wiremessage
)
