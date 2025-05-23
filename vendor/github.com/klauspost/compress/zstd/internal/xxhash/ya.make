GO_LIBRARY()

LICENSE(
    Apache-2.0 AND
    BSD-3-Clause AND
    MIT
)

VERSION(v1.18.0)

SRCS(
    xxhash.go
    xxhash_safe.go
)

GO_TEST_SRCS(xxhash_test.go)

IF (ARCH_X86_64)
    SRCS(
        xxhash_amd64.s
        xxhash_asm.go
    )
ENDIF()

IF (ARCH_ARM64)
    SRCS(
        xxhash_arm64.s
        xxhash_asm.go
    )
ENDIF()

IF (OS_LINUX AND ARCH_ARM7)
    SRCS(
        xxhash_other.go
    )
ENDIF()

END()

RECURSE(
    gotest
)
