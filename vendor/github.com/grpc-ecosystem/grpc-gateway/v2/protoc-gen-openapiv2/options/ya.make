PROTO_LIBRARY()

LICENSE(BSD-3-Clause)

VERSION(v2.19.1)

ONLY_TAGS(
    GO_PROTO
    CPP_PROTO
    JAVA_PROTO
    DESC_PROTO
    PY3_PROTO
)

IF (GO_PROTO)
    SRCS(
        annotations.pb.go
        openapiv2.pb.go
    )

    ADDINCL(
        GLOBAL
        FOR
        proto
        vendor/github.com/grpc-ecosystem/grpc-gateway/v2
    )
ELSE()
    PEERDIR(contrib/libs/googleapis-common-protos)

    PROTO_NAMESPACE(
        GLOBAL
        vendor/github.com/grpc-ecosystem/grpc-gateway/v2
    )

    GRPC()

    PY_NAMESPACE(protoc_gen_openapiv2.options)

    SRCS(
        annotations.proto
        openapiv2.proto
    )
ENDIF()

END()
