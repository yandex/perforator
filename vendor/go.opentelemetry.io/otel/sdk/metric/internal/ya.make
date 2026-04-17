GO_LIBRARY()

LICENSE(
    Apache-2.0 AND
    BSD-3-Clause
)

VERSION(v1.40.0)

SRCS(
    reuse_slice.go
)

END()

RECURSE(
    aggregate
    observ
    reservoir
    x
)
