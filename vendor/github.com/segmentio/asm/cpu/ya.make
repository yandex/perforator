GO_LIBRARY()

LICENSE(MIT)

VERSION(v1.2.0)

SRCS(
    cpu.go
)

GO_XTEST_SRCS(cpu_test.go)

END()

RECURSE(
    arm
    arm64
    cpuid
    gotest
    x86
)
