GO_LIBRARY()

LICENSE(
    Apache-2.0 AND
    BSD-3-Clause AND
    MIT
)

VERSION(v1.18.0)

SRCS(
    bitreader.go
    bitwriter.go
    compress.go
    decompress.go
    huff0.go
)

GO_TEST_SRCS(
    compress_test.go
    decompress_test.go
    fuzz_test.go
)

IF (ARCH_X86_64)
    SRCS(
        decompress_amd64.go
        decompress_amd64.s
    )
ENDIF()

IF (ARCH_ARM64)
    SRCS(
        decompress_generic.go
    )
ENDIF()

IF (OS_LINUX AND ARCH_ARM7)
    SRCS(
        decompress_generic.go
    )
ENDIF()

END()

RECURSE(
    gotest
)
