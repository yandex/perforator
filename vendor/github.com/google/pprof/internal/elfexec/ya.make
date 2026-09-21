GO_LIBRARY()

LICENSE(Apache-2.0)

VERSION(v0.0.0-20260111202518-71be6bfdd440)

SRCS(
    elfexec.go
)

GO_TEST_SRCS(elfexec_test.go)

END()

RECURSE(
    gotest
)
