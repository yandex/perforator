GO_LIBRARY()

LICENSE(Apache-2.0)

VERSION(v1.87.0)

SRCS(
    change_stream.pb.go
    commit_response.pb.go
    keys.pb.go
    location.pb.go
    mutation.pb.go
    query_plan.pb.go
    result_set.pb.go
    spanner.pb.go
    spanner_grpc.pb.go
    transaction.pb.go
    type.pb.go
)

END()
