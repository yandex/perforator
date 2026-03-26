GO_LIBRARY()

PEERDIR(
    perforator/agent/collector/progs
    vendor/github.com/cilium/ebpf
    ${GOSTD}/errors
    ${GOSTD}/unsafe
)

RUN_PROGRAM(
    perforator/ebpf/tools/btf2go
    --ignore profiler_state
    --elf unwinder.release.elf
    --elf unwinder.debug.elf
    --elf unwinder.release.php.elf
    --elf unwinder.debug.php.elf
    --elf unwinder.release.k54.php.elf
    --elf unwinder.debug.k54.php.elf
    --elf unwinder.release.k54.elf
    --elf unwinder.debug.k54.elf
    --package
    unwinder
    --output
    ${BINDIR}/unwinder.go
    CWD
    ${ARCADIA_BUILD_ROOT}/perforator/agent/collector/progs
    IN
    ${ARCADIA_BUILD_ROOT}/perforator/agent/collector/progs/unwinder.release.elf
    ${ARCADIA_BUILD_ROOT}/perforator/agent/collector/progs/unwinder.debug.elf
    ${ARCADIA_BUILD_ROOT}/perforator/agent/collector/progs/unwinder.release.k54.elf
    ${ARCADIA_BUILD_ROOT}/perforator/agent/collector/progs/unwinder.debug.k54.elf
    ${ARCADIA_BUILD_ROOT}/perforator/agent/collector/progs/unwinder.release.php.elf
    ${ARCADIA_BUILD_ROOT}/perforator/agent/collector/progs/unwinder.debug.php.elf
    ${ARCADIA_BUILD_ROOT}/perforator/agent/collector/progs/unwinder.release.k54.php.elf
    ${ARCADIA_BUILD_ROOT}/perforator/agent/collector/progs/unwinder.debug.k54.php.elf
    OUT
    ${BINDIR}/unwinder.go
)

RESOURCE(
    perforator/agent/collector/progs/unwinder.release.elf ebpf/unwinder.release.elf
    perforator/agent/collector/progs/unwinder.debug.elf ebpf/unwinder.debug.elf
    perforator/agent/collector/progs/unwinder.release.k54.elf ebpf/unwinder.release.k54.elf
    perforator/agent/collector/progs/unwinder.debug.k54.elf ebpf/unwinder.debug.k54.elf
    perforator/agent/collector/progs/unwinder.release.php.elf ebpf/unwinder.release.php.elf
    perforator/agent/collector/progs/unwinder.debug.php.elf ebpf/unwinder.debug.php.elf
    perforator/agent/collector/progs/unwinder.release.k54.php.elf ebpf/unwinder.release.k54.php.elf
    perforator/agent/collector/progs/unwinder.debug.k54.php.elf ebpf/unwinder.debug.k54.php.elf
)

SRCS(
    loader.go
)

END()
