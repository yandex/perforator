GO_LIBRARY()

LICENSE(
    Apache-2.0 AND
    BSD-3-Clause
)

VERSION(v1.42.0)

SRCS(
    attribute_group.go
    doc.go
    error_type.go
    exception.go
    schema.go
)

GO_TEST_SRCS(error_type_test.go)

END()

RECURSE(
    azureconv
    cicdconv
    containerconv
    cpuconv
    dbconv
    dnsconv
    faasconv
    genaiconv
    goconv
    gotest
    httpconv
    hwconv
    k8sconv
    messagingconv
    otelconv
    processconv
    rpcconv
    signalrconv
    systemconv
    vcsconv
)
