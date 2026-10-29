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
    build/platform/nodejs/22.23.4
    build/external_resources/pnpm
)

RUN_PYTHON3(
    ${CURDIR}/build.py
        --curdir ${CURDIR}
        --bindir ${BINDIR}
        --node-dir $NODEJS_22_23_4_RESOURCE_GLOBAL
        --pnpm-dir $PNPM_RESOURCE_GLOBAL
    IN
        ${UI_FILES}
    STDOUT ${BINDIR}/stdout
    OUT
        ${BINDIR}/output.tar
        ${BINDIR}/viewer.js
)

END()
