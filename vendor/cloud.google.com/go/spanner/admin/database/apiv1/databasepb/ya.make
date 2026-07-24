GO_LIBRARY()

LICENSE(Apache-2.0)

VERSION(v1.88.0)

SRCS(
    backup.pb.go
    backup_schedule.pb.go
    common.pb.go
    spanner_database_admin.pb.go
    spanner_database_admin_grpc.pb.go
)

END()
