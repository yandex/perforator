GO_LIBRARY()

LICENSE(BSD-3-Clause)

VERSION(v0.47.0)

SRCS(
    doc.go
)

GO_TEST_SRCS(
    recording_client_test.go
    recording_server_test.go
    recording_test.go
    server_test.go
    sshcli_test.go
    testdata_test.go
)

IF (OS_LINUX)
    GO_TEST_SRCS(
        agent_unix_test.go
        cert_test.go
        dial_unix_test.go
        forward_unix_test.go
        multi_auth_test.go
        session_test.go
        test_unix_test.go
    )
ENDIF()

IF (OS_DARWIN)
    GO_TEST_SRCS(
        agent_unix_test.go
        cert_test.go
        dial_unix_test.go
        forward_unix_test.go
        session_test.go
        test_unix_test.go
    )
ENDIF()

IF (OS_ANDROID)
    GO_TEST_SRCS(
        agent_unix_test.go
        cert_test.go
        dial_unix_test.go
        forward_unix_test.go
        multi_auth_test.go
        session_test.go
        test_unix_test.go
    )
ENDIF()

END()

RECURSE(
    #gotest
)
