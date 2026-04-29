GO_LIBRARY()

LICENSE(
    Apache-2.0 AND
    BSD-3-Clause
)

VERSION(v1.42.0)

SRCS(
    doc.go
    text_map_carrier.go
    text_map_propagator.go
)

GO_TEST_SRCS(
    text_map_carrier_test.go
    text_map_propagator_test.go
)

END()

RECURSE(
    gotest
)
