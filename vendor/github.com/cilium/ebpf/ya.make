GO_LIBRARY()

LICENSE(MIT)

VERSION(v0.17.1)

# requires root

GO_SKIP_TESTS(
    TestCollectionSpecRewriteMaps
    TestHaveMapMutabilityModifiers
    TestHaveProgTestRun
    TestIterateEmptyMap
    TestIterateMapInMap
    TestLink
    TestLoadCollectionSpec
    TestLoadRawTracepoint
    TestMap
    TestMapClose
    TestMapFreeze
    TestMapGetNextID
    TestMapInMap
    TestMapInMapValueSize
    TestMapIterate
    TestMapPin
    TestMapQueue
    TestNewMapFromID
    TestNewMapInMapFromFD
    TestNewProgramFromID
    TestPerfEventArray
    TestProgramAlter
    TestProgramBenchmark
    TestProgramClose
    TestProgramFromFD
    TestProgramGetNextID
    TestProgramKernelVersion
    TestProgramMarshaling
    TestProgramName
    TestProgramPin
    TestProgramRun
    TestProgramTestRunInterrupt
    TestProgramVerifierOutput
    TestProgramVerifierOutputOnError
    TestPerCPUMarshaling
    TestPerCPUMarshaling/LRUCPUHash
    TestMapContents
)

SRCS(
    attachtype_string.go
    collection.go
    cpu.go
    doc.go
    elf_reader.go
    elf_sections.go
    info.go
    linker.go
    map.go
    marshalers.go
    memory.go
    prog.go
    syscalls.go
    types.go
    types_string.go
    variable.go
)

END()

RECURSE(
    asm
    btf
    cmd
    # docs
    # examples
    features
    # gotest
    internal
    link
    perf
    pin
    ringbuf
    rlimit
)
