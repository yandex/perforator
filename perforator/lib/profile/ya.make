LIBRARY()

SRCS(
    builder.cpp
    flat_diffable.cpp
    merge.cpp
    merge_manager.cpp
    parallel_merge.cpp
    pprof.cpp
    profile.cpp
    validate.cpp
    visitor.cpp
    well_known.cpp
)

PEERDIR(
    perforator/proto/pprofprofile
    perforator/proto/profile

    library/cpp/containers/absl
    library/cpp/containers/stack_vector
    library/cpp/introspection
    library/cpp/json
    library/cpp/protobuf/inplace
    library/cpp/threading/future
)

END()

RECURSE(
    trie
)

RECURSE_FOR_TESTS(ut)
