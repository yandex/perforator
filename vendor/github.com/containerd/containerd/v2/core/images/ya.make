GO_LIBRARY()

LICENSE(Apache-2.0)

VERSION(v2.2.0)

SRCS(
    annotations.go
    diffid.go
    handlers.go
    image.go
    importexport.go
    mediatypes.go
)

GO_TEST_SRCS(image_test.go)

END()

RECURSE(
    archive
    gotest
    imagetest
    usage
)
