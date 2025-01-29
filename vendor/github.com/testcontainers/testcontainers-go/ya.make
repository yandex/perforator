GO_LIBRARY()

LICENSE(MIT)

VERSION(v0.31.0)

SRCS(
    config.go
    container.go
    docker.go
    docker_auth.go
    docker_client.go
    docker_mounts.go
    file.go
    generic.go
    image.go
    lifecycle.go
    logconsumer.go
    logger.go
    mounts.go
    network.go
    options.go
    parallel.go
    port_forwarding.go
    provider.go
    reaper.go
    testcontainers.go
    testing.go
)

GO_TEST_SRCS(
    # container_file_test.go
    # container_ignore_test.go
    # docker_auth_test.go
    # docker_client_test.go
    # docker_exec_test.go
    # docker_test.go
    # file_test.go
    # from_dockerfile_test.go
    # generic_test.go
    # image_substitutors_test.go
    # image_test.go
    # lifecycle_test.go
    # logconsumer_test.go
    # logger_test.go
    # parallel_test.go
    # provider_test.go
    # reaper_test.go
    # testcontainers_test.go
    # testing_test.go
)

GO_XTEST_SRCS(
    # config_test.go
    # container_test.go
    # docker_files_test.go
    # mounts_test.go
    # options_test.go
    # port_forwarding_test.go
    # testhelpers_test.go
)

END()

RECURSE(
    exec
    gotest
    internal
    network
    wait
)
