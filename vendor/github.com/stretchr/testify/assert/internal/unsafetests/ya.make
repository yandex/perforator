GO_LIBRARY()

LICENSE(MIT)

VERSION(v1.10.0)

SRCS(
    doc.go
)

GO_XTEST_SRCS(unsafetests_test.go)

END()

RECURSE(
    gotest
)
