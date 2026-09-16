GO_LIBRARY()

SRCS(
    models.go
)

END()

RECURSE(
    aggregated
    combined
    factory
    generations
    gc
)
