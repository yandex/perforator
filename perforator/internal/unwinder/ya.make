GO_LIBRARY()

PEERDIR(
    perforator/agent/collector/progs
    vendor/github.com/cilium/ebpf
    ${GOSTD}/errors
    ${GOSTD}/unsafe
)

RUN_PROGRAM(
    perforator/ebpf/tools/btf2go
    -elf
    perforator/agent/collector/progs/unwinder.debug.elf
    -package
    unwinder
    -output
    unwinder.go
    IN
    perforator/agent/collector/progs/unwinder.debug.elf
    OUT
    unwinder.go
)

RESOURCE(
    perforator/agent/collector/progs/unwinder.release.elf ebpf/unwinder.release.elf
    perforator/agent/collector/progs/unwinder.debug.elf ebpf/unwinder.debug.elf
    perforator/agent/collector/progs/unwinder.release.jvm.elf ebpf/unwinder.release.jvm.elf
    perforator/agent/collector/progs/unwinder.debug.jvm.elf ebpf/unwinder.debug.jvm.elf
    perforator/agent/collector/progs/unwinder.release.php.elf ebpf/unwinder.release.php.elf
    perforator/agent/collector/progs/unwinder.debug.php.elf ebpf/unwinder.debug.php.elf
)

SRCS(
    loader.go
)

END()
