GO_LIBRARY()

LICENSE(Apache-2.0)

VERSION(v0.0.0-20260111202518-71be6bfdd440)

SRCS(
    addr2liner.go
    addr2liner_llvm.go
    addr2liner_nm.go
    binutils.go
    disasm.go
)

GO_TEST_SRCS(
    binutils_test.go
    disasm_test.go
)

END()

RECURSE(
    # gotest
)
