GO_LIBRARY()

LICENSE(BSD-3-Clause)

VERSION(v0.45.0)

SRCS(
    bimport.go
    exportdata.go
    gcimporter.go
    iexport.go
    iimport.go
    predeclared.go
    support.go
    ureader.go
)

END()

RECURSE(
    # gotest # st/YMAKE-102
)
