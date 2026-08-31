GO_LIBRARY()

LICENSE(GPL-2.0)

SRCS(
    bookmark.go
    conf.go
    method_name.go
    scan_symbolization.go
    scan.go
    unsigned5.go
)

GO_TEST_SRCS(unsigned5_test.go)

END()

RECURSE(
    gotest
)
