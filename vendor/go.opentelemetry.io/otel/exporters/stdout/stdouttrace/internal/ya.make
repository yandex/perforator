GO_LIBRARY()

LICENSE(
    Apache-2.0 AND
    BSD-3-Clause
)

VERSION(v1.41.0)

SRCS(
    gen.go
    version.go
)

END()

RECURSE(
    counter
    observ
    x
)
