LIBRARY()

ADDINCL(
    ${ARCADIA_BUILD_ROOT}/contrib/libs/llvm18/lib/Target/X86
)

SRCS(
    analyze.cpp
    ehframe.cpp
)

PEERDIR(
    contrib/libs/llvm18/include
    contrib/libs/llvm18/lib/DebugInfo/DWARF
    contrib/libs/llvm18/lib/DebugInfo/Symbolize
    contrib/libs/llvm18/lib/Target
    # contrib/libs/llvm18/lib/Target/AArch64
    # contrib/libs/llvm18/lib/Target/ARM
    # contrib/libs/llvm18/lib/Target/BPF
    # contrib/libs/llvm18/lib/Target/LoongArch
    # contrib/libs/llvm18/lib/Target/NVPTX
    # contrib/libs/llvm18/lib/Target/PowerPC
    # contrib/libs/llvm18/lib/Target/WebAssembly
    contrib/libs/llvm18/lib/Target/X86
    perforator/agent/preprocessing/proto/parse
    perforator/agent/preprocessing/proto/python
    perforator/agent/preprocessing/proto/tls
    perforator/agent/preprocessing/proto/unwind
    perforator/lib/python
    perforator/lib/tls/parser
    perforator/lib/llvmex
    library/cpp/iterator
    library/cpp/streams/zstd
)

END()

RECURSE(go)
