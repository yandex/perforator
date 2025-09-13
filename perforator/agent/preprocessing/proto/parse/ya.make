PROTO_LIBRARY()

INCLUDE_TAGS(GO_PROTO)

PEERDIR(
    perforator/agent/preprocessing/proto/tls
    perforator/agent/preprocessing/proto/unwind
    perforator/agent/preprocessing/proto/pthread
    perforator/agent/preprocessing/proto/python
    perforator/agent/preprocessing/proto/php
)

SRCS(
    parse.proto
)

END()
