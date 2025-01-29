LIBRARY()

VERSION(1.2.3)

WITHOUT_LICENSE_TEXTS()

LICENSE(BSD-3-Clause)

SET(MUSL no)

NO_RUNTIME()

SRCS(
    dlvsym.cpp
)

IF (ARCH_X86_64)
    PEERDIR(
        contrib/libs/asmlib
        contrib/libs/asmglibc
    )
ENDIF()

PEERDIR(
    contrib/libs/musl
)

END()
