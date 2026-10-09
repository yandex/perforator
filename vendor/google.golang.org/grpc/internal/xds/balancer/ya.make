GO_LIBRARY()

LICENSE(Apache-2.0)

VERSION(v1.83.2)

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
