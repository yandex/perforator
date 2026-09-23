PROGRAM()

SRCS(main.cpp)

PEERDIR(
    contrib/libs/llvm22/include
    contrib/libs/llvm22/lib/Object
    perforator/lib/tls/parser
    perforator/lib/llvmex
)

END()
