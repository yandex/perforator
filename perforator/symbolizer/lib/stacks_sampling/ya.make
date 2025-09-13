LIBRARY()

PEERDIR(
    perforator/proto/pprofprofile

    perforator/symbolizer/lib/utils
)

SRCS(
    aggregating_sampler.cpp
    stacks_sampling_c.cpp
)

END()
