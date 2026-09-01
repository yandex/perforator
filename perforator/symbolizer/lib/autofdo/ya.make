LIBRARY()

PEERDIR(
    contrib/libs/llvm18/lib/Object

    perforator/lib/llvmex
    perforator/lib/profile
    perforator/lib/profile/c
    perforator/proto/profile

    contrib/libs/fmt

    library/cpp/yt/compact_containers
    library/cpp/containers/absl
)

SRCS(
    autofdo_c.cpp
    autofdo_input_builder.cpp
)

END()
