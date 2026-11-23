GO_LIBRARY()

LICENSE(Apache-2.0)

VERSION(v0.37.2)

SRCS(
    common.go
    dependentrequired.go
    discriminator.go
    doc.go
    each.go
    enum.go
    equality.go
    errors.go
    immutable.go
    item.go
    limits.go
    monotonic.go
    options.go
    required.go
    strfmt.go
    subfield.go
    testing.go
    union.go
    update.go
    zeroorone.go
)

GO_TEST_SRCS(
    dependentrequired_test.go
    discriminator_test.go
    each_test.go
    enum_test.go
    errors_test.go
    immutable_test.go
    item_test.go
    limits_test.go
    monotonic_test.go
    required_test.go
    strfmt_test.go
    testing_test.go
    union_test.go
    update_test.go
    util_test.go
    zeroorone_test.go
)

END()

RECURSE(
    constraints
    content
    gotest
)
