GO_LIBRARY()

LICENSE(Apache-2.0)

VERSION(v1.73.0)

SRCS(
    service.go
)

GO_TEST_SRCS(service_test.go)

IF (OS_LINUX AND ARCH_X86_64)
    GO_TEST_SRCS(service_sktopt_test.go)
ENDIF()

END()

RECURSE(
    gotest
)
