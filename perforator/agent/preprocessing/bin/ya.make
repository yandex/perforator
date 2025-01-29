PROGRAM(preprocessing)

SRCS(
    main.cpp
)

PEERDIR(
    contrib/libs/llvm18/include
    contrib/libs/llvm18/lib/DebugInfo/DWARF
    contrib/libs/llvm18/lib/DebugInfo/Symbolize
    contrib/libs/llvm18/lib/Target
    # contrib/libs/llvm18/lib/Target/AArch64
    # contrib/libs/llvm18/lib/Target/ARM
    # contrib/libs/llvm18/lib/Target/BPF
    # contrib/libs/llvm18/lib/Target/NVPTX
    # contrib/libs/llvm18/lib/Target/PowerPC
    contrib/libs/llvm18/lib/Target/X86
    library/cpp/streams/zstd
    perforator/agent/preprocessing/lib
    perforator/agent/preprocessing/proto/parse
    perforator/agent/preprocessing/proto/tls
    perforator/agent/preprocessing/proto/unwind
    perforator/lib/llvmex
)

END()
