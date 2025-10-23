LIBRARY()

PEERDIR(
    perforator/lib/demangle
    perforator/proto/pprofprofile
    perforator/symbolizer/lib/gsym

    library/cpp/int128
)

SRCS(
    cluster_top_c.cpp
    service_perf_top_aggregator.cpp
)

END()
