GO_LIBRARY()

LICENSE(BSD-3-Clause)

VERSION(v0.35.1-0.20250728180453-01a3475a31bc)

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
)
