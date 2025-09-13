GO_LIBRARY()

LICENSE(Apache-2.0)

VERSION(v0.5.2)

SRCS(
    input.go
)

END()

RECURSE(
    reflect_output
    source_output
)
