GO_LIBRARY()

LICENSE(Apache-2.0)

VERSION(v0.37.2)

SRCS(
    doc.go
    register.go
    types.go
    zz_generated.deepcopy.go
)

END()

RECURSE(
    fuzzer
    install
    v1
)
