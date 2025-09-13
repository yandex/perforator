GO_LIBRARY()

LICENSE(MIT)

VERSION(v0.0.0-20180802200727-47ae307949d0)

SRCS(
    builder.go
    reflect.go
    registry.go
)

GO_XTEST_SRCS(
    builder_test.go
    example_test.go
)

END()

RECURSE(
    gotest
)
