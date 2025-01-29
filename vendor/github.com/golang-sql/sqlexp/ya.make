GO_LIBRARY()

LICENSE(BSD-3-Clause)

VERSION(v0.1.0)

GO_SKIP_TESTS(TestMSSQL)

SRCS(
    doc.go
    messages.go
    mssql.go
    namer.go
    pg.go
    querier.go
    quoter.go
    registry.go
    savepoint.go
)

END()
