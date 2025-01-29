GO_LIBRARY()

LICENSE(MIT)

VERSION(v0.11.1)

SRCS(
    clip.go
    geometry.go
    layer.go
    marshal.go
    projection.go
    simplify.go
    unmarshal.go
)

GO_TEST_SRCS(
    clip_test.go
    geometry_test.go
    marshal_test.go
    projection_test.go
    simplify_test.go
)

GO_XTEST_SRCS(example_test.go)

END()

RECURSE(
    gotest
    vectortile
)
