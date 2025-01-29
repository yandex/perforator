GO_LIBRARY()

LICENSE(BSD-3-Clause)

VERSION(v0.21.0)

SRCS(
    bitfield.go
)

GO_TEST_SRCS(
    bitfield_test.go
    gen1_test.go
    gen2_test.go
)

END()

RECURSE(
    gotest
)
