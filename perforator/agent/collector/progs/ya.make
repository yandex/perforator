GEN_LIBRARY()

BUILD_ONLY_IF(WARNING LINUX)

PEERDIR(
    perforator/lib/tls
)

SET(BPF_FLAGS
    -O2
    --debug
    -mcpu=v3
    -D__TARGET_ARCH_x86
    -D__x86_64__
    -D__KERNEL__
    -Wall
    -Werror
)

BPF(unwinder/unwinder.c unwinder.release.elf $BPF_FLAGS)
BPF(unwinder/unwinder.c unwinder.debug.elf $BPF_FLAGS -DBPF_DEBUG)

ADDINCL(
    contrib/libs/libbpf/include
    contrib/libs/linux-headers
    perforator/ebpf/include
)

END()
