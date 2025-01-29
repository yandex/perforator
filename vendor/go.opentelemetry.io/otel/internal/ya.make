GO_LIBRARY()

LICENSE(Apache-2.0)

VERSION(v1.33.0)

SRCS(
    gen.go
    rawhelpers.go
)

END()

RECURSE(
    attribute
    baggage
    global
    internaltest
    matchers
)
