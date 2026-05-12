GO_LIBRARY()

LICENSE(Apache-2.0)

VERSION(v0.33.0)

SRCS(
    meta.go
    yaml.go
)

GO_TEST_SRCS(
    meta_test.go
    yaml_test.go
)

END()

RECURSE(
    gotest
)
