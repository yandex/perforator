GO_PROGRAM()

LICENSE(MIT)

VERSION(v0.17.1)

SRCS(
    doc.go
    flags.go
    main.go
    makedep.go
    tools.go
)

END()

RECURSE(
    gen
    # gotest
    internal
    test
)
