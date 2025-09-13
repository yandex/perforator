GO_LIBRARY()

LICENSE(Apache-2.0)

VERSION(v0.5.2)

SRCS(
    bugreport.go
    bugreport_mock.go
)

END()

RECURSE(
    faux
)
