PROTO_LIBRARY()

INCLUDE_TAGS(GO_PROTO)

PEERDIR(
    perforator/agent/preprocessing/proto/tls
    perforator/agent/preprocessing/proto/unwind
    perforator/agent/preprocessing/proto/python
)

SRCS(
    parse.proto
)

END()
