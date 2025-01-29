PROGRAM()

SRCS(main.cpp)

PEERDIR(
    contrib/libs/llvm18/include
    contrib/libs/llvm18/lib/Object
    perforator/lib/tls/parser
    perforator/lib/llvmex
)

END()
