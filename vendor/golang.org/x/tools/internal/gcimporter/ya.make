GO_LIBRARY()

LICENSE(BSD-3-Clause)

VERSION(v0.35.1-0.20250728180453-01a3475a31bc)

SRCS(
    bimport.go
    exportdata.go
    gcimporter.go
    iexport.go
    iimport.go
    predeclared.go
    support.go
    ureader_yes.go
)

END()

RECURSE(
    # gotest # st/YMAKE-102
)
