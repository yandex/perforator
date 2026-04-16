GO_LIBRARY()

LICENSE(BSD-3-Clause)

VERSION(v1.68.0)

IF (OS_LINUX AND ARCH_X86_64)
    SRCS(
        capi_linux_amd64.go
        poll_linux_amd64.go
    )
ENDIF()

IF (OS_LINUX AND ARCH_ARM64)
    SRCS(
        capi_linux_arm64.go
        poll_linux_arm64.go
    )
ENDIF()

IF (OS_LINUX AND ARCH_ARM6 OR OS_LINUX AND ARCH_ARM7)
    SRCS(
        capi_linux_arm.go
        poll_linux_arm.go
    )
ENDIF()

IF (OS_DARWIN AND ARCH_X86_64)
    SRCS(
        capi_darwin_amd64.go
        poll_darwin_amd64.go
    )
ENDIF()

IF (OS_DARWIN AND ARCH_ARM64)
    SRCS(
        capi_darwin_arm64.go
        poll_darwin_arm64.go
    )
ENDIF()

IF (OS_ANDROID)
    SRCS(
        capi_linux_arm64.go
        poll_linux_arm64.go
    )
ENDIF()

END()
