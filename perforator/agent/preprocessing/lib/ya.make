LIBRARY()

SRCS(
    analyze.cpp
    ehframe.cpp
    sframe.cpp
    unwind_table_builder.cpp
)


PEERDIR(
    contrib/libs/llvm22/include
    contrib/libs/llvm22/lib/DebugInfo/DWARF
    contrib/libs/llvm22/lib/DebugInfo/Symbolize
    contrib/libs/llvm22/lib/Target
    contrib/libs/llvm22/lib/Target/AArch64
    contrib/libs/llvm22/lib/Target/ARM
    # contrib/libs/llvm22/lib/Target/BPF
    # contrib/libs/llvm22/lib/Target/LoongArch
    # contrib/libs/llvm22/lib/Target/NVPTX
    # contrib/libs/llvm22/lib/Target/PowerPC
    # contrib/libs/llvm22/lib/Target/WebAssembly
    contrib/libs/llvm22/lib/Target/X86
    contrib/libs/llvm22/lib/Object
    perforator/agent/preprocessing/proto/parse
    perforator/agent/preprocessing/proto/python
    perforator/agent/preprocessing/proto/tls
    perforator/agent/preprocessing/proto/unwind
    perforator/internal/linguist/jvm/analysis/lite
    perforator/lib/pthread
    perforator/lib/python
    perforator/lib/php
    perforator/lib/lua
    perforator/lib/tls/parser
    perforator/lib/llvmex
    library/cpp/iterator
    library/cpp/streams/zstd
)

IF (ARCH_AARCH64)

PEERDIR(
    contrib/libs/llvm22/lib/Target/AArch64
    contrib/libs/llvm22/lib/Target/AArch64/Disassembler
)

ENDIF()

END()

RECURSE(go)
