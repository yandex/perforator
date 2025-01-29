GO_LIBRARY()

LICENSE(MIT)

VERSION(v0.31.0)

SRCS(
    bootstrap.go
    client.go
    docker_host.go
    docker_rootless.go
    docker_socket.go
    images.go
    labels.go
)

GO_TEST_SRCS(
    docker_host_test.go
    docker_rootless_test.go
    images_test.go
)

END()

RECURSE(
    gotest
    network
)
