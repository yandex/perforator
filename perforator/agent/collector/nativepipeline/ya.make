LIBRARY()

SRCS(
    raw_sample.cpp
    runtime.cpp
    state.cpp
    wire.cpp
)

PEERDIR(
    contrib/libs/linux-headers
    perforator/lib/profile/c
)

END()

RECURSE_FOR_TESTS(
    ut
)
