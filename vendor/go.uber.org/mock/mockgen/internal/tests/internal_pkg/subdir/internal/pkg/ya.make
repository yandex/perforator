GO_LIBRARY()

LICENSE(Apache-2.0)

VERSION(v0.6.0)

SRCS(
    input.go
)

END()

RECURSE(
    reflect_output
    source_output
)
