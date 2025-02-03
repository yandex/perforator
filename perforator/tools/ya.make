IF (NOT OPENSOURCE)
    RECURSE(
        gsym_vs_dwarf
    )
ENDIF()

RECURSE(
    cpu_burner
    lbr_to_autofdo
    pprofconvert
)
