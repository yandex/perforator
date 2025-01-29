GO_LIBRARY()

IF(OPENSOURCE)
    SRCS(
        maxprocs.go
    )
ELSE()
    SRCS(
        maxprocs_yandex.go
    )
ENDIF()



END()
