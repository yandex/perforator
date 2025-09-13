LIBRARY()

PEERDIR(
    perforator/symbolizer/lib/gsym

    perforator/proto/pprofprofile

    library/cpp/int128
)

SRCS(
    cluster_top_c.cpp
    service_perf_top_aggregator.cpp
)

END()
