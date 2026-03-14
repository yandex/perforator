GO_LIBRARY()

LICENSE(BSD-3-Clause)

VERSION(v0.0.0-20251113190631-e25ba8c21ef6)

SRCS(
    maps.go
)

GO_TEST_SRCS(maps_test.go)

END()

RECURSE(
    gotest
)
