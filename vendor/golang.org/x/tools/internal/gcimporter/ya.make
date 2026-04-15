GO_LIBRARY()

LICENSE(BSD-3-Clause)

VERSION(v0.42.0)

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
