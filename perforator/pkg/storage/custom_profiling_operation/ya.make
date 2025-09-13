GO_LIBRARY()

SRCS(
    models.go
)

END()

RECURSE(
    mocks
    postgres
)
