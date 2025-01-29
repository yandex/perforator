GO_LIBRARY()

LICENSE(BSD-3-Clause)

VERSION(v0.21.0)

SRCS(
    encoding.go
)

GO_XTEST_SRCS(
    encoding_test.go
    example_test.go
)

END()

RECURSE(
    charmap
    gotest
    htmlindex
    ianaindex
    internal
    japanese
    korean
    simplifiedchinese
    traditionalchinese
    unicode
)
