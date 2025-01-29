GO_LIBRARY()

LICENSE(Apache-2.0)

VERSION(v2.18.0)

SRCS(
    meta.go
)

GO_EMBED_PATTERN(meta.yml)

END()
