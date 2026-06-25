GO_LIBRARY()

LICENSE(Apache-2.0)

VERSION(v0.9.0)

SRCS(
    attestations.go
    certconstraint.go
    envelope.go
    hashlib.go
    keylib.go
    match.go
    model.go
    rulelib.go
    runlib.go
    util.go
    verifylib.go
)

GO_TEST_SRCS(
    attestations_test.go
    certconstraint_test.go
    envelope_test.go
    examples_test.go
    in_toto_test.go
    keylib_test.go
    match_test.go
    model_test.go
    rulelib_test.go
    runlib_test.go
    util_test.go
    verifylib_test.go
)

IF (OS_LINUX)
    SRCS(
        util_unix.go
    )
ENDIF()

IF (OS_DARWIN)
    SRCS(
        util_unix.go
    )
ENDIF()

IF (OS_WINDOWS)
    SRCS(
        util_windows.go
    )
ENDIF()

IF (OS_ANDROID)
    SRCS(
        util_unix.go
    )
ENDIF()

IF (OS_EMSCRIPTEN)
    SRCS(
        util_unix.go
    )
ENDIF()

END()

RECURSE(
    gotest
    slsa_provenance
)
