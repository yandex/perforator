LIBRARY()

PEERDIR(
    contrib/libs/llvm18/lib/Object

    perforator/proto/pprofprofile
    perforator/lib/llvmex

    contrib/libs/fmt

    library/cpp/yt/compact_containers
    library/cpp/containers/absl_flat_hash
)

SRCS(
    autofdo_c.cpp
    autofdo_input_builder.cpp
)

END()
