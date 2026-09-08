LIBRARY()

# According to lawyers, this effectively depends on OpenJDK which is licensed under GPL v2
LICENSE(GPL-2.0)


SRCS(
    offsets.cpp
    parser.cpp
    static_analysis.cpp
)

PEERDIR(
    perforator/internal/linguist/jvm/analysis/offset_registry
)

PEERDIR(
    contrib/libs/llvm18/include
    contrib/libs/llvm18/lib/Target/X86
    
    perforator/lib/llvmex
)

END()
