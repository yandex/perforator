GO_LIBRARY()

TAG(ya:run_go_benchmark)
IF (NOT OPENSOURCE)
    DATA(
        sbr://3766900569=stacks
        sbr://9066593741=pprof-a
        sbr://9066590432=pprof-b
    )
ENDIF()

SRCS(
    blocks.go
    hsv.go
    render.go
    strtab.go
    text_format.go
)
IF (NOT OPENSOURCE)
    GO_TEST_SRCS(
        blocks_test.go
        render_json_test.go
    )
    GO_XTEST_SRCS(render_test.go)
ENDIF()

GO_TEST_SRCS(text_format_test.go)

GO_EMBED_PATTERN(tmpl.html)
GO_EMBED_PATTERN(new_templ.html)

RESOURCE(
    ${ARCADIA_BUILD_ROOT}/perforator/ui/union/viewer.js viewer.js
)

PEERDIR(perforator/ui/union)

END()

IF (NOT OPENSOURCE)
    RECURSE(gotest)
ENDIF()

RECURSE(
    cmd
    format
)
