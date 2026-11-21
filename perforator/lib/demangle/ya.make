LIBRARY()

PEERDIR(
    contrib/libs/icu
    contrib/libs/llvm22/lib/Demangle
    contrib/libs/re2
    library/cpp/containers/stack_vector
)

SRCS(
    cython.cpp
    demangle.cpp
    itanium.cpp
    rustc.cpp
)

END()

RECURSE_FOR_TESTS(ut)
