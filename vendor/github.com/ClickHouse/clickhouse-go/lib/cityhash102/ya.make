GO_LIBRARY()

LICENSE(MIT)

VERSION(v1.5.4)

SRCS(
    city64.go
    cityhash.go
    doc.go
)

GO_TEST_SRCS(cityhash_test.go)

END()

RECURSE(
    gotest
)
