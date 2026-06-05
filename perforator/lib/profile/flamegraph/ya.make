LIBRARY()

SRCS(
    build.cpp
    render.cpp
)

PEERDIR(
    library/cpp/iterator

    perforator/lib/profile
    perforator/lib/profile/trie
    perforator/proto/profile

    contrib/libs/rapidjson
)

END()
