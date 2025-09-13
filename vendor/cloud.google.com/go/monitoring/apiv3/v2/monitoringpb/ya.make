GO_LIBRARY()

LICENSE(Apache-2.0)

VERSION(v1.24.1)

SRCS(
    alert.pb.go
    alert_service.pb.go
    common.pb.go
    dropped_labels.pb.go
    group.pb.go
    group_service.pb.go
    metric.pb.go
    metric_service.pb.go
    mutation_record.pb.go
    notification.pb.go
    notification_service.pb.go
    query_service.pb.go
    service.pb.go
    service_service.pb.go
    snooze.pb.go
    snooze_service.pb.go
    span_context.pb.go
    uptime.pb.go
    uptime_service.pb.go
)

END()
