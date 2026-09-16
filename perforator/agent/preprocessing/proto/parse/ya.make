PROTO_LIBRARY()

INCLUDE(${ARCADIA_ROOT}/perforator/proto/tags.inc)

PEERDIR(
    perforator/agent/preprocessing/proto/jvm
    perforator/agent/preprocessing/proto/tls
    perforator/agent/preprocessing/proto/unwind
    perforator/agent/preprocessing/proto/pthread
    perforator/agent/preprocessing/proto/python
    perforator/agent/preprocessing/proto/php
    perforator/agent/preprocessing/proto/lua
)

SRCS(
    parse.proto
)

END()
