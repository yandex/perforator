GO_LIBRARY()

SRCS(
    buildid.go
    buildinfo.go
    elf_note.go
    textbits.go
)

END()

RECURSE(
    cmd
)
