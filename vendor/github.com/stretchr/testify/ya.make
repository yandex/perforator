GO_LIBRARY()

SUBSCRIBER(g:go-contrib)

LICENSE(MIT)

VERSION(v1.10.0)

SRCS(
    doc.go
)

GO_TEST_SRCS(package_test.go)

END()

RECURSE(
    assert
    gotest
    http
    mock
    require
    #suite
)
