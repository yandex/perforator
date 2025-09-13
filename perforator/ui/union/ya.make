SUBSCRIBER(g:perforator)

UNION()

DECLARE_IN_DIRS(
    UI
    *
    SRCDIR ${ARCADIA_ROOT}/perforator/ui
    DIRS .
    RECURSIVE
)

PEERDIR(
    build/platform/nodejs/20.18.1
    build/external_resources/pnpm/10.14.0
)

RUN_PYTHON3(
    ${CURDIR}/build.py
        --curdir ${CURDIR}
        --bindir ${BINDIR}
        --node-dir $NODEJS_20_18_1_RESOURCE_GLOBAL
        --pnpm-dir $PNPM_10_14_0_RESOURCE_GLOBAL
    IN
        ${UI_FILES}
    STDOUT ${BINDIR}/stdout
    OUT
        ${BINDIR}/output.tar
        ${BINDIR}/viewer.js
)

END()
