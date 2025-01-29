GO_LIBRARY()

LICENSE(MIT)

VERSION(v0.17.1)

SRCS(
    doc.go
    map.go
    misc.go
    prog.go
    version.go
)

END()

RECURSE(
    # gotest
)
