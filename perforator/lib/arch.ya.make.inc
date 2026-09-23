# Incls.
IF (ARCH_X86_64)
    ADDINCL(
        ${ARCADIA_BUILD_ROOT}/contrib/libs/llvm22/lib/Target/X86
    )
ELSEIF(ARCH_AARCH64)
    ADDINCL(
        ${ARCADIA_BUILD_ROOT}/contrib/libs/llvm22/lib/Target/AArch64
    )
ENDIF()

# Peerdirs.
IF (ARCH_X86_64)
    PEERDIR(
        contrib/libs/llvm22/lib/Target/X86
    )
ELSEIF(ARCH_AARCH64)
    PEERDIR(
        contrib/libs/llvm22/lib/Target/AArch64
    )
ENDIF()
