GTEST()

PEERDIR(
    perforator/symbolizer/lib/cluster_top
    perforator/lib/profile
    perforator/proto/pprofprofile

    library/cpp/int128
)

SRCS(
    top_aggregator_ut.cpp
)

END()
