LIBRARY()

IF (ARCH_X86_64)

PEERDIR(
    perforator/lib/lua/asm/x86
)

ELSEIF(ARCH_AARCH64)

PEERDIR(
    perforator/lib/lua/asm/arm
)

ENDIF()

END()
