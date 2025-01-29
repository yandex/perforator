GO_LIBRARY()

LICENSE(BSD-3-Clause)

VERSION(v0.22.1-0.20240829175637-39126e24d653)

SRCS(
    bimport.go
    exportdata.go
    gcimporter.go
    iexport.go
    iimport.go
    newInterface11.go
    support_go118.go
    unified_no.go
    ureader_yes.go
)

END()

RECURSE(
    # gotest # st/YMAKE-102
)
