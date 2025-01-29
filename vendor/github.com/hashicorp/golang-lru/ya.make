GO_LIBRARY()

LICENSE(MPL-2.0)

VERSION(v1.0.2)

SRCS(
    2q.go
    arc.go
    doc.go
    lru.go
    testing.go
)

GO_TEST_SRCS(
    2q_test.go
    arc_test.go
    lru_test.go
)

END()

RECURSE(
    gotest
    simplelru
)
