GEN_LIBRARY()

BUILD_ONLY_IF(WARNING LINUX)

PEERDIR(
    perforator/lib/tls
)

SET(BPF_FLAGS
    -O2
    --debug
    -mcpu=v3
    -D__KERNEL__
    -Wall
    -Werror
)

IF (ARCH_X86_64)
    SET(BPF_ARCH_FLAGS
        -D__TARGET_ARCH_x86
        -D__x86_64__
    )
ELSEIF (ARCH_AARCH64)
    SET(BPF_ARCH_FLAGS
        -D__TARGET_ARCH_arm64
        -D__aarch64__
    )
ENDIF()

BPF(unwinder/unwinder.bpf.c unwinder.release.elf $BPF_FLAGS $BPF_ARCH_FLAGS)
BPF(unwinder/unwinder.bpf.c unwinder.debug.elf $BPF_FLAGS $BPF_ARCH_FLAGS -DBPF_DEBUG)

BPF(unwinder/unwinder.bpf.c unwinder.release.k54.elf $BPF_FLAGS $BPF_ARCH_FLAGS -DPERFORATOR_COMPAT_5_4 -Wno-unused-function)
BPF(unwinder/unwinder.bpf.c unwinder.debug.k54.elf $BPF_FLAGS $BPF_ARCH_FLAGS -DBPF_DEBUG -DPERFORATOR_COMPAT_5_4 -Wno-unused-function)

BPF(unwinder/unwinder.bpf.c unwinder.release.php.elf $BPF_FLAGS $BPF_ARCH_FLAGS -DPERFORATOR_ENABLE_PHP)
BPF(unwinder/unwinder.bpf.c unwinder.debug.php.elf $BPF_FLAGS $BPF_ARCH_FLAGS -DBPF_DEBUG -DPERFORATOR_ENABLE_PHP)

BPF(unwinder/unwinder.bpf.c unwinder.release.k54.php.elf $BPF_FLAGS $BPF_ARCH_FLAGS -DPERFORATOR_COMPAT_5_4 -Wno-unused-function -DPERFORATOR_ENABLE_PHP)
BPF(unwinder/unwinder.bpf.c unwinder.debug.k54.php.elf $BPF_FLAGS $BPF_ARCH_FLAGS -DBPF_DEBUG -DPERFORATOR_COMPAT_5_4 -Wno-unused-function -DPERFORATOR_ENABLE_PHP)


ADDINCL(
    contrib/libs/libbpf/include
    contrib/libs/linux-headers
    perforator/ebpf/include
)

END()

RECURSE_FOR_TESTS(
    tests
)
