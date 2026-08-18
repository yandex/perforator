GO_LIBRARY()

SRCS(
    opts.go
    storage.go
)

END()

RECURSE(
    fs
    models
    s3
)
