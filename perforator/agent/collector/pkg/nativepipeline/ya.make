GO_LIBRARY()

PEERDIR(
    perforator/agent/collector/nativepipeline
    perforator/lib/profile/c
)

CGO_SRCS(
    runtime_cgo.go
    state_cgo.go
)

SRCS(
    process_listener.go
    runtime.go
)

END()
