GO_LIBRARY()

LICENSE(BSD-3-Clause)

VERSION(v0.0.0-20250305212735-054e65f0b394)

SRCS(
    maps.go
)

GO_TEST_SRCS(maps_test.go)

END()

RECURSE(
    gotest
)
