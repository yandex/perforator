GO_LIBRARY()

LICENSE(MIT)

VERSION(v4.15.2)

SRCS(
    log.go
    migrate.go
    migration.go
    util.go
)

END()

RECURSE(
    cli
    cmd
    database
    internal
    source
)
