GO_LIBRARY()

LICENSE(Apache-2.0)

VERSION(v1.17.1)

SRCS(
    assume_role_provider.go
    ec2_provider.go
    ecs_provider.go
    env_provider.go
    imds_provider.go
    static_provider.go
)

END()
