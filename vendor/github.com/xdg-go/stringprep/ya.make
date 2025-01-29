GO_LIBRARY()

LICENSE(Apache-2.0)

VERSION(v1.0.4)

SRCS(
    bidi.go
    doc.go
    error.go
    map.go
    profile.go
    saslprep.go
    set.go
    tables.go
)

GO_TEST_SRCS(
    map_test.go
    profile_test.go
    saslprep_test.go
    set_test.go
)

GO_XTEST_SRCS(examples_test.go)

END()

RECURSE(
    gotest
)
