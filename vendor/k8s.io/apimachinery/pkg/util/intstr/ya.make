GO_LIBRARY()

LICENSE(Apache-2.0)

VERSION(v0.26.1)

SRCS(
    generated.pb.go
    instr_fuzz.go
    intstr.go
)

GO_TEST_SRCS(intstr_test.go)

END()

RECURSE(
    gotest
)
