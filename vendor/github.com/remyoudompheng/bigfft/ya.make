GO_LIBRARY()

LICENSE(BSD-3-Clause)

VERSION(v0.0.0-20230129092748-24d4a6f8daec)

SRCS(
    arith_decl.go
    fermat.go
    fft.go
    scan.go
)

GO_TEST_SRCS(
    calibrate_test.go
    fermat_test.go
    fft_test.go
    scan_test.go
)

END()

RECURSE(
    gotest
)
