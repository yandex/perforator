GO_LIBRARY()

LICENSE(BSD-3-Clause)

VERSION(v0.42.0)

SRCS(
    classify_call.go
    element.go
    errorcode.go
    errorcode_string.go
    fx.go
    isnamed.go
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
