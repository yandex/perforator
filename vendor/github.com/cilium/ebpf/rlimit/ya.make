GO_LIBRARY()

LICENSE(MIT)

VERSION(v0.17.1)

GO_SKIP_TESTS(TestRemoveMemlock)

SRCS(
    doc.go
)

IF (OS_LINUX)
    SRCS(
        rlimit_linux.go
    )
ENDIF()

IF (OS_DARWIN)
    SRCS(
        rlimit_other.go
    )
ENDIF()

IF (OS_WINDOWS)
    SRCS(
        rlimit_other.go
    )
ENDIF()

END()
