LIBRARY()

SRCS(tls.cpp)

PEERDIR(
    contrib/libs/llvm18/include
    contrib/libs/llvm18/lib/Demangle
    contrib/libs/llvm18/lib/Object
)

END()
