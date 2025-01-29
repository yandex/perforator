GO_LIBRARY()

LICENSE(Apache-2.0)

VERSION(v1.63.2)

SRCS(
    backoff.go
    balancer_wrapper.go
    call.go
    clientconn.go
    codec.go
    dialoptions.go
    doc.go
    interceptor.go
    picker_wrapper.go
    pickfirst.go
    preloader.go
    resolver_wrapper.go
    rpc_util.go
    server.go
    service_config.go
    shared_buffer_pool.go
    stream.go
    trace.go
    trace_withtrace.go
    version.go
)

GO_TEST_SRCS(
    balancer_wrapper_test.go
    clientconn_authority_test.go
    clientconn_parsed_target_test.go
    clientconn_test.go
    codec_test.go
    default_dial_option_server_option_test.go
    grpc_test.go
    picker_wrapper_test.go
    resolver_test.go
    rpc_util_test.go
    server_test.go
    service_config_test.go
    shared_buffer_pool_test.go
    trace_test.go
)

GO_XTEST_SRCS(
    resolver_balancer_ext_test.go
    server_ext_test.go
    stream_test.go
)

END()

RECURSE(
    admin
    attributes
    authz
    backoff
    balancer
    benchmark
    binarylog
    channelz
    codes
    connectivity
    credentials
    encoding
    experimental
    # gotest
    grpclog
    health
    internal
    interop
    keepalive
    metadata
    orca
    peer
    profiling
    reflection
    resolver
    serviceconfig
    stats
    status
    tap
    # test
    testdata
    xds
)
