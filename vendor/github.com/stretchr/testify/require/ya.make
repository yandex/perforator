GO_LIBRARY()

LICENSE(MIT)

VERSION(v1.10.0)

SRCS(
    doc.go
    forward_requirements.go
    require.go
    require_forward.go
    requirements.go
)

GO_TEST_SRCS(
    forward_requirements_test.go
    requirements_test.go
)

END()

RECURSE(
    gotest
)
