GO_LIBRARY()

LICENSE(Apache-2.0)

VERSION(v1.73.0)

SRCS(
    balancer.go
)

END()

RECURSE(
    cdsbalancer
    clusterimpl
    clustermanager
    clusterresolver
    loadstore
    outlierdetection
    priority
    wrrlocality
)
