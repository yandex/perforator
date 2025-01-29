GO_LIBRARY()

SRCS(
    models.go
)

END()

RECURSE(
    compound
    inmemory
    postgres
)
