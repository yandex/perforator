LIBRARY()

SRCS(
    clock.cpp
    profile_accumulator.cpp
    raw_sample.cpp
    runtime.cpp
    signals.cpp
    state.cpp
    wire.cpp
)

PEERDIR(
    contrib/libs/linux-headers
    library/cpp/int128
    perforator/lib/profile/c
)

END()

RECURSE_FOR_TESTS(
    ut
)
