GO_LIBRARY()

LICENSE(MIT)

VERSION(v0.0.0-20240606120523-5a60cdf6a761)

SRCS(
    pgservicefile.go
)

GO_XTEST_SRCS(pgservicefile_test.go)

END()

RECURSE(
    gotest
)
