GO_LIBRARY()

PEERDIR(
    perforator/ebpf/examples/04-offcpu-time/prog
    vendor/github.com/cilium/ebpf
)

RUN_PROGRAM(
    perforator/agent/collector/cmd/btf2go
    -elf
    perforator/ebpf/examples/04-offcpu-time/prog/prog.debug.elf
    -package
    loader
    -output
    prog.go
    IN
    perforator/ebpf/examples/04-offcpu-time/prog/prog.debug.elf
    OUT
    prog.go
)

RESOURCE(
    perforator/ebpf/examples/04-offcpu-time/prog/prog.release.elf ebpf/prog.release.elf
    perforator/ebpf/examples/04-offcpu-time/prog/prog.debug.elf ebpf/prog.debug.elf
)

SRCS(
    loader.go
)

END()
