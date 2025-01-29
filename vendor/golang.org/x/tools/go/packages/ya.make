GO_LIBRARY()

LICENSE(BSD-3-Clause)

VERSION(v0.22.1-0.20240829175637-39126e24d653)

SRCS(
    doc.go
    external.go
    golist.go
    golist_overlay.go
    loadmode_string.go
    packages.go
    visit.go
)

END()

RECURSE(
    gopackages
    internal
    packagestest
)
