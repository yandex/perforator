GO_LIBRARY()

LICENSE(Apache-2.0)

VERSION(v0.18.2)

SRCS(
    appdefaults.go
)

IF (OS_LINUX)
    SRCS(
        appdefaults_linux.go
        appdefaults_unix.go
    )
ENDIF()

IF (OS_DARWIN)
    SRCS(
        appdefaults_unix.go
        appdefaults_unix_nolinux.go
    )
ENDIF()

IF (OS_WINDOWS)
    SRCS(
        appdefaults_windows.go
    )
ENDIF()

IF (OS_ANDROID)
    SRCS(
        appdefaults_linux.go
        appdefaults_unix.go
    )
ENDIF()

IF (OS_EMSCRIPTEN)
    SRCS(
        appdefaults_unix.go
    )
ENDIF()

END()
