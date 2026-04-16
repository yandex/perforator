GO_LIBRARY()

LICENSE(BSD-3-Clause)

VERSION(v0.0.0-20260218203240-3dfff04db8fa)

SRCS(
    maps.go
)

GO_TEST_SRCS(maps_test.go)

END()

RECURSE(
    gotest
)
