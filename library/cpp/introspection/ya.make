LIBRARY()

SRCS(
    hash_ops.cpp
    introspection.cpp
)

PEERDIR(
    contrib/libs/pfr
)

END()

RECURSE_FOR_TESTS(tests)
