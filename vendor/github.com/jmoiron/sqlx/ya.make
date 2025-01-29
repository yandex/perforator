GO_LIBRARY()

LICENSE(MIT)

VERSION(v1.3.5)

# DISABLED due to go-sqlite3 dependency being broken
#GO_TEST_SRCS(
#    named_context_test.go
#    named_test.go
#    sqlx_context_test.go
#    sqlx_test.go
#)

SRCS(
    bind.go
    doc.go
    named.go
    named_context.go
    sqlx.go
    sqlx_context.go
)

GO_TEST_SRCS(
    bind_test.go
    named_context_test.go
    named_test.go
    sqlx_context_test.go
    sqlx_test.go
)

END()

RECURSE(
    # gotest DISABLED due to go-sqlite3 dependency being broken
    reflectx
    types
)
