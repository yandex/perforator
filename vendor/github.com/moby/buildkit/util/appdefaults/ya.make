GO_LIBRARY()

LICENSE(Apache-2.0)

VERSION(v0.13.2)

SRCS(
    appdefaults.go
)

IF (OS_LINUX)
    SRCS(
        appdefaults_unix.go
    )
ENDIF()

IF (OS_DARWIN)
    SRCS(
        appdefaults_unix.go
    )
ENDIF()

IF (OS_WINDOWS)
    SRCS(
        appdefaults_windows.go
    )
ENDIF()

IF (OS_ANDROID)
    SRCS(
        appdefaults_unix.go
    )
ENDIF()

IF (OS_EMSCRIPTEN)
    SRCS(
        appdefaults_unix.go
    )
ENDIF()

END()
