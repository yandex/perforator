GO_LIBRARY()

LICENSE(MIT)

VERSION(v1.27.0)

SRCS(
    level_enabler.go
)

END()

RECURSE(
    bufferpool
    color
    exit
    pool
    readme
    stacktrace
    ztest
)
