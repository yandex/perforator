GO_LIBRARY()

LICENSE(Apache-2.0)

VERSION(v0.16.0)

IF (OS_LINUX)
    SRCS(
        avc_cache_stats.go
        avc_hash_stats.go
        fs.go
    )

    GO_TEST_SRCS(
        # avc_cache_stats_test.go
        # avc_hash_stats_test.go
        # fs_test.go
    )
ENDIF()

END()

RECURSE(
    gotest
)
