GO_LIBRARY()

LICENSE(BSD-3-Clause)

VERSION(v1.2.0)

SRCS(
    fe.go
    fe_extra.go
    fe_generic.go
)

GO_TEST_SRCS(
    fe_alias_test.go
    fe_bench_test.go
    fe_extra_test.go
    fe_test.go
)

IF (ARCH_X86_64)
    SRCS(
        fe_amd64.go
        fe_amd64.s
    )
ENDIF()

IF (ARCH_ARM64)
    SRCS(
        fe_amd64_noasm.go
    )
ENDIF()

IF (OS_LINUX AND ARCH_ARM6 OR OS_LINUX AND ARCH_ARM7)
    SRCS(
        fe_amd64_noasm.go
    )
ENDIF()

IF (OS_EMSCRIPTEN)
    SRCS(
        fe_amd64_noasm.go
    )
ENDIF()

END()

RECURSE(
    gotest
)
