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
    ${ARCADIA_ROOT}/perforator/agent/tests/perfbuf_bench_test.go
)

DEPENDS(
    ${ARCADIA_ROOT}/perforator/agent/tests/dummies/perfbuf_bench
)

SIZE(MEDIUM)

END()
