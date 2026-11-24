GO_LIBRARY()

SRCS(
    config.go
    kubelet.go
    pod.go
    source.go
    transport.go
)

GO_TEST_SRCS(
    kubelet_test.go
    pods_lister_test.go
    source_test.go
)

GO_TEST_EMBED_PATTERN(kubelet-configz-response.json)

END()

RECURSE(
    gotest
)
