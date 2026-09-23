PROGRAM(pythonparse)

INCLUDE(${ARCADIA_ROOT}/perforator/lib/arch.ya.make.inc)

SRCS(main.cpp)

PEERDIR(
    contrib/libs/llvm22/include
    contrib/libs/llvm22/lib/Object

    perforator/lib/python
    perforator/lib/llvmex
)

END()
