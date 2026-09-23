LIBRARY()

PEERDIR(
    contrib/libs/llvm22/lib/Demangle
    contrib/libs/re2
)

SRCS(
    demangle.cpp
    itanium.cpp
    rustc.cpp
)

END()

RECURSE_FOR_TESTS(ut)
