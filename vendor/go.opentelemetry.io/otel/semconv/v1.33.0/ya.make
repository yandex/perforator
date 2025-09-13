GO_LIBRARY()

LICENSE(Apache-2.0)

VERSION(v1.37.0)

SRCS(
    attribute_group.go
    doc.go
    exception.go
    schema.go
)

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
