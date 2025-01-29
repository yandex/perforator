GO_LIBRARY()

LICENSE(MIT)

VERSION(v4.15.2)

SRCS(
    go-bindata.go
)

END()

RECURSE(
    examples
    testdata
)
