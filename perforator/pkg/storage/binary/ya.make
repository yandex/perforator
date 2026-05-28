GO_LIBRARY()

SRCS(
    config.go
    models.go
    storage.go
)

END()

RECURSE(
    compound
    decompressing
    meta
    mock
)
