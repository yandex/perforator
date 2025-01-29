LIBRARY()

PEERDIR(
    contrib/libs/llvm18/lib/DebugInfo/Symbolize
    contrib/libs/llvm18/lib/DebugInfo/GSYM
    contrib/libs/llvm18/lib/DebugInfo/DWARF
    contrib/libs/llvm18/lib/Object

    contrib/libs/fmt

    library/cpp/yt/compact_containers

    perforator/lib/llvmex
)

SRCS(
    gsym.cpp
    gsym_symbolizer.cpp
)

END()
