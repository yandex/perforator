GO_LIBRARY()

LICENSE(BSD-3-Clause)

VERSION(v0.35.1-0.20250728180453-01a3475a31bc)

SRCS(
    classify_call.go
    element.go
    errorcode.go
    errorcode_string.go
    qualifier.go
    recv.go
    toonew.go
    types.go
    varkind.go
    zerovalue.go
)

END()

RECURSE(
    typeindex
)
