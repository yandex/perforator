GO_LIBRARY()

LICENSE(MIT)

VERSION(v4.4.0+incompatible)

SRCS(
    codec.go
    generator.go
    sql.go
    uuid.go
)

GO_TEST_SRCS(
    codec_test.go
    generator_test.go
    sql_test.go
    uuid_test.go
)

END()

RECURSE(
    gotest
)
