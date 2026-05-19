GO_LIBRARY()

LICENSE(MIT)

VERSION(v0.12.0)

SRCS(
    set.go
    tile.go
)

GO_TEST_SRCS(tile_test.go)

END()

RECURSE(
    gotest
    tilecover
)
