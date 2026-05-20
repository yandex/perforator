GO_LIBRARY()

LICENSE(MIT)

VERSION(v0.32.0)

SRCS(
    version.go
)

END()

RECURSE(
    config
    core
)
