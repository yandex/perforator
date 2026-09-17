GO_TEST()

TAG(ya:run_go_benchmark ya:not_autocheck)

IF(CI)
    TAG(ya:manual)
ENDIF()

SRCS(
    ${ARCADIA_ROOT}/perforator/agent/tests/metrics.go
    ${ARCADIA_ROOT}/perforator/agent/tests/util.go
)

GO_TEST_SRCS(
    ${ARCADIA_ROOT}/perforator/agent/tests/sample_reader_test.go
)

SIZE(MEDIUM)

END()
