GO_LIBRARY()

LICENSE(Apache-2.0)

VERSION(v1.46.7)

SRCS(
    accesspoint_arn.go
    arn.go
    outpost_arn.go
    s3_object_lambda_arn.go
)

GO_TEST_SRCS(
    accesspoint_arn_test.go
    arn_test.go
    outpost_arn_test.go
)

END()

RECURSE(
    gotest
)
