GO_LIBRARY()

LICENSE(Apache-2.0)

VERSION(v1.73.0)

SRCS(
    orcalb.go
    soak_tests.go
    test_utils.go
)

END()

RECURSE(
    alts
    client
    fake_grpclb
    grpc_testing
    http2
    server
    stress
    xds_federation
    # yo
)

IF (OS_LINUX)
    RECURSE(
        grpclb_fallback
    )
ENDIF()
