GTEST()

SRCS(
    hash_ops_ut.cpp
    member_get_ut.cpp
    members_count_ut.cpp
    members_ut.cpp
)

PEERDIR(
    library/cpp/introspection
    library/cpp/introspection/tests
)

END()
