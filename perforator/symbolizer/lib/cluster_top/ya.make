RECURSE_FOR_TESTS(result_ut)

LIBRARY()

PEERDIR(
    perforator/lib/demangle
    perforator/lib/profile
    perforator/lib/profile/c
    perforator/symbolizer/lib/gsym

    library/cpp/int128
)

SRCS(
    cluster_top_c.cpp
    perf_top_aggregator.cpp
    top_aggregator_session.cpp
)

END()

RECURSE(
    ut
)
