GO_LIBRARY()

SRCS(
    config.go
    discoverer.go
    dns.go
    newdiscoverer.go
    predefined.go
)

IF (NOT OPENSOURCE)
    SRCS(
        yp.go
    )
ENDIF()

END()
