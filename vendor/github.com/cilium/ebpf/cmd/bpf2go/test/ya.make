GO_LIBRARY()

LICENSE(MIT)

VERSION(v0.17.3)

GO_SKIP_TESTS(TestLoadingObjects)

SRCS(
    doc.go
)

IF (OS_LINUX)
    SRCS(
        test_bpfel.go
    )
ENDIF()

IF (OS_ANDROID)
    SRCS(
        test_bpfel.go
    )
ENDIF()

GO_EMBED_PATTERN(test_bpfel.o)

END()
