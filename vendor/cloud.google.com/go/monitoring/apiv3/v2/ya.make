GO_LIBRARY()

LICENSE(Apache-2.0)

VERSION(v1.24.1)

SRCS(
    alert_policy_client.go
    auxiliary.go
    auxiliary_go123.go
    doc.go
    group_client.go
    helpers.go
    metric_client.go
    notification_channel_client.go
    query_client.go
    service_monitoring_client.go
    snooze_client.go
    uptime_check_client.go
    version.go
)

GO_XTEST_SRCS(
    alert_policy_client_example_go123_test.go
    alert_policy_client_example_test.go
    group_client_example_go123_test.go
    group_client_example_test.go
    metric_client_example_go123_test.go
    metric_client_example_test.go
    notification_channel_client_example_go123_test.go
    notification_channel_client_example_test.go
    query_client_example_go123_test.go
    query_client_example_test.go
    service_monitoring_client_example_go123_test.go
    service_monitoring_client_example_test.go
    snooze_client_example_go123_test.go
    snooze_client_example_test.go
    uptime_check_client_example_go123_test.go
    uptime_check_client_example_test.go
)

END()

RECURSE(
    gotest
    monitoringpb
)
