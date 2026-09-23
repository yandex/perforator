LIBRARY()

# According to lawyers, this effectively depends on OpenJDK which is licensed under GPL v2
LICENSE(GPL-2.0)

SRCS(
    analyzer_impl.cpp
    offset_registry.cpp
)

PEERDIR(
    contrib/libs/llvm22/include
    contrib/libs/llvm22/lib/Target/X86
    
    perforator/lib/llvmex

    perforator/internal/linguist/jvm/analysis/api
)

END()
