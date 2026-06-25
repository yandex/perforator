GO_LIBRARY()

LICENSE(MIT)

VERSION(v0.17.3)

IF (OS_LINUX)
    SRCS(
        poller.go
    )
ENDIF()

IF (OS_ANDROID)
    SRCS(
        poller.go
    )
ENDIF()

END()
