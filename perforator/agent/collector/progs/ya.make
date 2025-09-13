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

BPF(unwinder/unwinder.bpf.c unwinder.release.elf $BPF_FLAGS)
BPF(unwinder/unwinder.bpf.c unwinder.debug.elf $BPF_FLAGS -DBPF_DEBUG)
BPF(unwinder/unwinder.bpf.c unwinder.release.jvm.elf $BPF_FLAGS -DPERFORATOR_ENABLE_JVM)
BPF(unwinder/unwinder.bpf.c unwinder.debug.jvm.elf $BPF_FLAGS -DBPF_DEBUG -DPERFORATOR_ENABLE_JVM)
BPF(unwinder/unwinder.bpf.c unwinder.release.php.elf $BPF_FLAGS -DPERFORATOR_ENABLE_PHP)
BPF(unwinder/unwinder.bpf.c unwinder.debug.php.elf $BPF_FLAGS -DBPF_DEBUG -DPERFORATOR_ENABLE_PHP)


ADDINCL(
    contrib/libs/libbpf/include
    contrib/libs/linux-headers
    perforator/ebpf/include
)

END()

RECURSE_FOR_TESTS(
    tests
)
