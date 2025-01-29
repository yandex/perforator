GO_PROGRAM()

LICENSE(Apache-2.0)

VERSION(v1.63.2)

IF (OS_LINUX)
    SRCS(
        client_linux.go
    )
ENDIF()

END()
