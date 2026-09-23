LIBRARY()

SRCS(
    lite_analysis.cpp
)


ADDINCL(
    ${ARCADIA_BUILD_ROOT}/contrib/libs/llvm22/lib/Target/X86
)

PEERDIR(
    contrib/libs/llvm22/include
    
    perforator/lib/llvmex

    perforator/internal/linguist/jvm/analysis/api
)

END()
