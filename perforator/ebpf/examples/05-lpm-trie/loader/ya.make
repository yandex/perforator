GO_LIBRARY()

PEERDIR(
    perforator/ebpf/examples/05-lpm-trie/prog
    vendor/github.com/cilium/ebpf
    ${GOSTD}/errors
)

RUN_PROGRAM(
    perforator/agent/collector/cmd/btf2go
    -elf
    perforator/ebpf/examples/05-lpm-trie/prog/prog.debug.elf
    -package
    loader
    -output
    prog.go
    IN
    perforator/ebpf/examples/05-lpm-trie/prog/prog.debug.elf
    OUT
    prog.go
)

RESOURCE(
    perforator/ebpf/examples/05-lpm-trie/prog/prog.release.elf ebpf/prog.release.elf
    perforator/ebpf/examples/05-lpm-trie/prog/prog.debug.elf ebpf/prog.debug.elf
)

SRCS(
    loader.go
)

END()
