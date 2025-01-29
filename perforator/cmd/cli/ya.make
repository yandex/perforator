GO_PROGRAM(perforator)

IF (OS_LINUX)
    ALLOCATOR(TCMALLOC)
ENDIF()

SRCS(
    main.go
)

END()
