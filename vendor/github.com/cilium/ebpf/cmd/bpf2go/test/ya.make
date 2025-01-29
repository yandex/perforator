GO_LIBRARY()

LICENSE(MIT)

VERSION(v0.17.1)

GO_SKIP_TESTS(TestLoadingObjects)

SRCS(
    doc.go
    test_bpfel.go
)

GO_EMBED_PATTERN(test_bpfel.o)

END()
