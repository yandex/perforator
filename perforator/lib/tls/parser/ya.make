LIBRARY()

SRCS(tls.cpp)

PEERDIR(
    contrib/libs/llvm22/include
    contrib/libs/llvm22/lib/Demangle
    contrib/libs/llvm22/lib/Object
)

END()
