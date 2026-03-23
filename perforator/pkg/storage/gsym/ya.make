GO_LIBRARY()

SRCS(
    models.go
    storage.go
)

END()

RECURSE(
    compound
    meta
)
