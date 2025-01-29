GO_LIBRARY()

PEERDIR(
    perforator/proto/storage
    perforator/pkg/storage/profile/compound
)

SRCS(
    config.go
    sampler.go
    server.go
)

END()
