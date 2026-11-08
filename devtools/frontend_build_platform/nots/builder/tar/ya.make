PROGRAM(nots-tar)

ALLOCATOR(SYSTEM)

STRIP()

SRCS(
    main.cpp
)

PEERDIR(
    contrib/libs/libarchive
)

END()

RECURSE_FOR_TESTS(
    tests
)
