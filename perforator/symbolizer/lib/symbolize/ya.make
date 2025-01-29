LIBRARY()

PEERDIR(
    contrib/libs/llvm18/lib/DebugInfo/Symbolize
    contrib/libs/pdqsort
    contrib/libs/re2

    library/cpp/logger/global
    library/cpp/yt/compact_containers

    perforator/proto/pprofprofile
    perforator/lib/llvmex

    perforator/symbolizer/lib/gsym
)

SRCS(
    symbolizec.cpp
    symbolizec.h
    symbolizer.cpp
    symbolizer.h
)

END()
