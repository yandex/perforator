DLL()

PEERDIR(
    contrib/libs/jdk
    contrib/libs/llvm18/lib/Target

    perforator/lib/elf

    perforator/internal/linguist/jvm/unwind/lib
)

SRCS(
    jni.cpp
)

END()
