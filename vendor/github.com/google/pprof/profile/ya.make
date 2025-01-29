GO_LIBRARY()

LICENSE(Apache-2.0)

VERSION(v0.0.0-20240424215950-a892ee059fd6)

SRCS(
    encode.go
    filter.go
    index.go
    legacy_java_profile.go
    legacy_profile.go
    merge.go
    profile.go
    proto.go
    prune.go
)

GO_TEST_SRCS(
    filter_test.go
    index_test.go
    legacy_profile_test.go
    merge_test.go
    profile_test.go
    proto_test.go
    prune_test.go
)

END()

RECURSE(
    gotest
)
