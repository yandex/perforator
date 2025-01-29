GO_LIBRARY()

LICENSE(Apache-2.0)

VERSION(v1.46.7)

SRCS(
    chain_provider.go
    context_background_go1.7.go
    context_go1.9.go
    credentials.go
    env_provider.go
    shared_credentials_provider.go
    static_provider.go
)

GO_TEST_SRCS(
    chain_provider_test.go
    credentials_bench_test.go
    credentials_go1.7_test.go
    credentials_test.go
    env_provider_test.go
    shared_credentials_provider_test.go
    static_provider_test.go
)

END()

RECURSE(
    ec2rolecreds
    endpointcreds
    gotest
    plugincreds
    processcreds
    ssocreds
    stscreds
)
