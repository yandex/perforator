GO_LIBRARY()

LICENSE(MIT)

VERSION(v0.31.0)

SRCS(
    version.go
)

END()

RECURSE(
    config
    core
)
