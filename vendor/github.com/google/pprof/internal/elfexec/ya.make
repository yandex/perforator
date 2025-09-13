GO_LIBRARY()

LICENSE(Apache-2.0)

VERSION(v0.0.0-20250317173921-a4b03ec1a45e)

SRCS(
    elfexec.go
)

GO_TEST_SRCS(elfexec_test.go)

END()

RECURSE(
    gotest
)
