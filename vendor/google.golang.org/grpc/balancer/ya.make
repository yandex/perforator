GO_LIBRARY()

LICENSE(Apache-2.0)

VERSION(v1.79.1)

SRCS(
    balancer.go
    conn_state_evaluator.go
    subconn.go
)

GO_TEST_SRCS(conn_state_evaluator_test.go)

END()

RECURSE(
    base
    endpointsharding
    gotest
    grpclb
    lazy
    leastrequest
    pickfirst
    randomsubsetting
    ringhash
    rls
    roundrobin
    weightedroundrobin
    weightedtarget
)
