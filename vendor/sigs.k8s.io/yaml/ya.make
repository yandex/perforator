GO_LIBRARY()

LICENSE(
    Apache-2.0 AND
    BSD-3-Clause AND
    MIT
)

VERSION(v1.4.0)

SRCS(
    fields.go
    yaml.go
    yaml_go110.go
)

GO_TEST_SRCS(
    bench_test.go
    err_test.go
    yaml_go110_test.go
    yaml_test.go
)

END()

RECURSE(
    gotest
    goyaml.v2
    goyaml.v3
)
