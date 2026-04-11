GO_LIBRARY()

LICENSE(Apache-2.0)

VERSION(v0.0.0-20250403155104-27863c87afa6)

SRCS(
    proftest.go
)

GO_EMBED_PATTERN(testdata/large.cpu)

END()
