GO_LIBRARY()

PEERDIR(
    perforator/agent/collector/nativepipeline
    perforator/lib/profile/c
)

CGO_SRCS(state_cgo.go)

END()
