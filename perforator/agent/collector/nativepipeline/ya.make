LIBRARY()

SRCS(
    raw_sample.cpp
    runtime.cpp
    state.cpp
    wire.cpp
)

PEERDIR(
    library/cpp/cache
    library/cpp/containers/absl
    perforator/lib/profile/c
)

END()

RECURSE_FOR_TESTS(
    ut
)
