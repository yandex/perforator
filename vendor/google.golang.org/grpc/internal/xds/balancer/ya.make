GO_LIBRARY()

LICENSE(Apache-2.0)

VERSION(v1.80.0)

SRCS(
    balancer.go
)

END()

RECURSE(
    cdsbalancer
    clusterimpl
    clustermanager
    loadstore
    outlierdetection
    priority
    wrrlocality
)
