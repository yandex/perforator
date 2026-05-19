GO_LIBRARY()

LICENSE(Apache-2.0)

VERSION(v2.46.0)

SRCS(
    error_util.go
    json_helper.go
    package.go
    utils.go
)

END()

RECURSE(
    issues
    std
    stress
)
