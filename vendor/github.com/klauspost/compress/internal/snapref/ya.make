GO_LIBRARY()

LICENSE(
    Apache-2.0 AND
    BSD-3-Clause AND
    MIT
)

VERSION(v1.17.9)

SRCS(
    decode.go
    decode_other.go
    encode.go
    encode_other.go
    snappy.go
)

END()
