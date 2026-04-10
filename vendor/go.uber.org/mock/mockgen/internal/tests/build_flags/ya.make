GO_LIBRARY()

LICENSE(Apache-2.0)

VERSION(v0.6.0)

SRCS(
    directive.go
)

END()

RECURSE(
    mock1
    mock2
)
