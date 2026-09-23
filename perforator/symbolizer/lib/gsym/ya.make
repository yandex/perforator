LIBRARY()

PEERDIR(
    contrib/libs/llvm22/lib/DebugInfo/Symbolize
    contrib/libs/llvm22/lib/DebugInfo/GSYM
    contrib/libs/llvm22/lib/DebugInfo/DWARF
    contrib/libs/llvm22/lib/Object

    contrib/libs/fmt

    library/cpp/yt/compact_containers

    perforator/lib/elf
    perforator/lib/llvmex
)

SRCS(
    gsym.cpp
    gsym_symbolizer.cpp
)

END()
