GO_LIBRARY()

LICENSE(BSD-3-Clause)

VERSION(v0.176.1)

SRCS(
    cba.go
    conn_pool.go
    creds.go
    s2a.go
    settings.go
    version.go
)

GO_TEST_SRCS(
    cba_test.go
    # creds_test.go
    s2a_test.go
    settings_test.go
)

END()

RECURSE(
    cert
    gensupport
    gotest
    impersonate
    third_party
)
