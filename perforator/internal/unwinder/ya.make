GO_LIBRARY()

PEERDIR(
    perforator/agent/collector/progs
    vendor/github.com/cilium/ebpf
    ${GOSTD}/encoding/binary
    ${GOSTD}/errors
    ${GOSTD}/fmt
    ${GOSTD}/unsafe
)

RUN_PROGRAM(
    perforator/ebpf/tools/btf2go
    --ignore jvm_staging
    --elf unwinder.release.elf
    --elf unwinder.debug.elf
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
    OUT
    ${BINDIR}/unwinder.go
)

RESOURCE(
    perforator/agent/collector/progs/unwinder.release.elf ebpf/unwinder.release.elf
    perforator/agent/collector/progs/unwinder.debug.elf ebpf/unwinder.debug.elf
    perforator/agent/collector/progs/unwinder.release.k54.elf ebpf/unwinder.release.k54.elf
    perforator/agent/collector/progs/unwinder.debug.k54.elf ebpf/unwinder.debug.k54.elf
)

SRCS(
    loader.go
    sample_parsed.go
    sample_parser.go
)

GO_TEST_SRCS(sample_parser_test.go)

END()

RECURSE_FOR_TESTS(
    gotest
)
