LIBRARY()

SRCS(
    raw_sample.cpp
    state.cpp
    wire.cpp
)

PEERDIR(
    perforator/lib/profile/c
)

END()

RECURSE_FOR_TESTS(
    ut
)
