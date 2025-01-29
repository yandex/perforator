GEN_LIBRARY()

BUILD_ONLY_IF(WARNING LINUX)

SET(BPF_FLAGS
    -O2
    --debug
    -mcpu=v3
    -D__TARGET_ARCH_x86
    -D__x86_64__
    -D__KERNEL__
    -Wall
)

BPF(prog.c prog.release.elf $BPF_FLAGS)
BPF(prog.c prog.debug.elf $BPF_FLAGS -DBPF_DEBUG)

ADDINCL(
    contrib/libs/libbpf/include
    contrib/libs/linux-headers
    perforator/ebpf/include
)

END()
