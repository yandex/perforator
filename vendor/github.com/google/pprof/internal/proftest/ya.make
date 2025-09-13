GO_LIBRARY()

LICENSE(Apache-2.0)

VERSION(v0.0.0-20250317173921-a4b03ec1a45e)

SRCS(
    proftest.go
)

GO_EMBED_PATTERN(testdata/large.cpu)

END()
