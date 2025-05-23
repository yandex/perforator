GO_LIBRARY()

LICENSE(Apache-2.0)

VERSION(v2.33.1)

SRCS(
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
