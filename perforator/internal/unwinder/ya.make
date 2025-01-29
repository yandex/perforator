GO_LIBRARY()

PEERDIR(
    perforator/agent/collector/progs
    vendor/github.com/cilium/ebpf
    ${GOSTD}/errors
    ${GOSTD}/unsafe
)

RUN_PROGRAM(
    perforator/agent/collector/cmd/btf2go
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
)

SRCS(
    loader.go
)

END()
