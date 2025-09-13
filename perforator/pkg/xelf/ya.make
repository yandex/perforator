GO_LIBRARY()

SRCS(
    buildid.go
    buildinfo.go
    elf_note.go
    phdr.go
    symbol.go
    textbits.go
)

IF (NOT OPENSOURCE)
    GO_TEST_SRCS(
        symbol_test.go
    )
ENDIF()

END()

RECURSE(
    cmd
)

IF (NOT OPENSOURCE)
    RECURSE(
        gotest
    )
ENDIF()
