GO_LIBRARY()

LICENSE(MIT)

VERSION(v0.11.1)

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
