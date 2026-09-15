GO_LIBRARY()

LICENSE(
    Apache-2.0 AND
    BSD-3-Clause
)

VERSION(v1.46.0)

ALL_GO_SRCS()

GO_TEST_SRCS(error_type_test.go)

END()

RECURSE(
    azureconv
    cicdconv
    containerconv
    dbconv
    dnsconv
    faasconv
    goconv
    gotest
    httpconv
    hwconv
    k8sconv
    messagingconv
    nfsconv
    openshiftconv
    otelconv
    processconv
    rpcconv
    signalrconv
    systemconv
    vcsconv
)
