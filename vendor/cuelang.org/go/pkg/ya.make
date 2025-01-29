GO_LIBRARY()

LICENSE(Apache-2.0)

VERSION(v0.4.3)

SRCS(
    doc.go
    register.go
)

END()

RECURSE(
    crypto
    encoding
    gen
    html
    internal
    list
    math
    net
    path
    regexp
    strconv
    strings
    struct
    text
    time
    tool
    uuid
)
