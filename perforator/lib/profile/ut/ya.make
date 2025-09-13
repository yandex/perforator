GTEST()

REQUIREMENTS(ram:8)

SIZE(MEDIUM)

SRCS(
    builder_ut.cpp
    diff_ut.cpp
    merge_ut.cpp

    golden.cpp
)

PEERDIR(
    contrib/libs/re2
    perforator/lib/profile
)

END()
