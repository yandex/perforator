GO_LIBRARY()

LICENSE(MIT)

VERSION(v2.3.0)

SRCS(
    xxhash.go
    xxhash_asm.go
    xxhash_unsafe.go
)

GO_TEST_SRCS(
    bench_test.go
    xxhash_test.go
    xxhash_unsafe_test.go
)

IF (ARCH_X86_64)
    SRCS(
        xxhash_amd64.s
    )
ENDIF()

IF (ARCH_ARM64)
    SRCS(
        xxhash_arm64.s
    )
ENDIF()

END()

RECURSE(
    # gotest
    xxhsum
)

IF (OS_LINUX)
    RECURSE(
        #        dynamic
    )
ENDIF()

IF (OS_DARWIN)
    RECURSE(
        #        dynamic
    )
ENDIF()
