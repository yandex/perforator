GO_PROGRAM()

LICENSE(Apache-2.0)

VERSION(v0.5.2)

SRCS(
    deprecated.go
    generic.go
    gob.go
    mockgen.go
    package_mode.go
    parse.go
    version.go
)

END()

RECURSE(
    internal
    model
)
