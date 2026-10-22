GTEST()

REQUIREMENTS(ram:8)

SIZE(MEDIUM)

SRCS(
    builder_ut.cpp
    diff_ut.cpp
    merge_ut.cpp
    pprof_ut.cpp
    strtab_ut.cpp
    validate_ut.cpp
)

PEERDIR(
    perforator/lib/profile/ut/lib
)

END()

IF (NOT OPENSOURCE)
    RECURSE(
        yandex-specific
    )
ENDIF()
