GO_TEST()

LICENSE(Apache-2.0)

VERSION(v1.79.3)

GO_XTEST_SRCS(
    ads_stream_ack_nack_test.go
    ads_stream_backoff_test.go
    ads_stream_flow_control_test.go
    ads_stream_restart_test.go
    ads_stream_watch_test.go
    authority_test.go
    dump_test.go
    helpers_test.go
    lds_watchers_test.go
    metrics_test.go
    misc_watchers_test.go
)

END()
