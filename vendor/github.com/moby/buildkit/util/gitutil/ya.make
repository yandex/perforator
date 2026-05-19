GO_LIBRARY()

LICENSE(Apache-2.0)

VERSION(v0.13.2)

SRCS(
    git_cli.go
    git_cli_helpers.go
    git_ref.go
    git_url.go
)

GO_TEST_SRCS(
    git_ref_test.go
    git_url_test.go
)

END()

RECURSE(
    gotest
)
