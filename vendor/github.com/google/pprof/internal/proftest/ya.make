GO_LIBRARY()

LICENSE(Apache-2.0)

VERSION(v0.0.0-20260111202518-71be6bfdd440)

SRCS(
    proftest.go
)

GO_EMBED_PATTERN(testdata/large.cpu)

END()
