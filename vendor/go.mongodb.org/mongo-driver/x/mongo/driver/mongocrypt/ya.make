GO_LIBRARY()

LICENSE(Apache-2.0)

VERSION(v1.17.1)

SRCS(
    errors_not_enabled.go
    mongocrypt_context_not_enabled.go
    mongocrypt_kms_context_not_enabled.go
    mongocrypt_not_enabled.go
    state.go
)

END()

RECURSE(
    options
)
