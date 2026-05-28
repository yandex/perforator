GO_LIBRARY()

SRCS(
    handle.go
    opts.go
    storage.go
)

END()

RECURSE(
    fs
    models
    s3
)
