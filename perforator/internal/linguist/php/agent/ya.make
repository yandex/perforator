GO_LIBRARY()

GO_EMBED_PATTERN(offsets/*.json)

SRCS(
    offsets.go
    php.go
    stackprocessor.go
)

END()
